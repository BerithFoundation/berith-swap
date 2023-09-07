package bridge

import (
	"berith-swap/bridge/blockstore"
	"berith-swap/bridge/chain"
	"berith-swap/bridge/config"
	"berith-swap/bridge/contract"
	"berith-swap/bridge/message"
	"berith-swap/bridge/util"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
)

var (
	BlockRetryLimit           = 5
	BlockRetryInterval        = time.Second * 5
	DefaultBlockConfirmations = big.NewInt(10)
)

type SenderChain struct {
	c                  *chain.Chain
	msgChan            chan<- message.DepositMessage
	blockStore         *blockstore.Blockstore
	blockConfirmations *big.Int
	bridgeContract     *contract.BridgeContract
	startBlock         *big.Int
}

func NewSenderChain(ch chan<- message.DepositMessage, cfg *config.Config, idx int) *SenderChain {
	chain, err := chain.NewChain(cfg, idx)
	if err != nil {
		chain.Logger.Panic().Err(err).Msgf("cannot init chain. idx:%d", idx)
	}
	bs, err := blockstore.NewBlockstore(cfg.BlockStorePath, chain.Name)
	if err != nil {
		chain.Logger.Panic().Err(err).Msg("cannot init block store")
	}
	blockConfirmations, err := util.StringToBig(cfg.ChainConfig[idx].BlockConfirmations, 10) // contract 등록 유무로 isSender
	if err != nil {
		chain.Logger.Error().Msgf("cannot get block confirmations from config. set default:%d", DefaultBlockConfirmations.Int64())
		blockConfirmations = DefaultBlockConfirmations
	}

	startBlock, err := bs.TryLoadLatestBlock()
	if err != nil {
		chain.Logger.Panic().Err(err).Msgf("cannot load latest block from block store.")
	}
	if !cfg.IsLoaded {
		curr, err := chain.EvmClient.LatestBlock()
		if err != nil {
			chain.Logger.Error().Err(err).Msgf("cannot get latest block through evmclient.")
			return nil
		}
		startBlock = curr
	}

	sc := SenderChain{
		c:                  chain,
		msgChan:            ch,
		blockStore:         bs,
		blockConfirmations: blockConfirmations,
		startBlock:         startBlock,
	}

	sc.setSenderBridgeContract(&cfg.ChainConfig[idx])
	return &sc
}

func (s *SenderChain) setSenderBridgeContract(chainCfg *config.RawChainConfig) error {
	if chainCfg.BridgeAddress == "" {
		err := fmt.Errorf("sender chain dosen't have bridge contract address. Chain:%s, Idx:%d", s.c.Name, chainCfg.Idx)
		s.c.Logger.Error().Err(err).Msg("check config.json")
		return err
	}

	err := s.c.EvmClient.EnsureHasBytecode(common.HexToAddress(chainCfg.BridgeAddress))
	if err != nil {
		s.c.Logger.Panic().Err(err).Msgf("contract dosen't exist this chain url:%s", s.c.Endpoint)
	}

	c, err := contract.IniBridgeContract(s.c.EvmClient, chainCfg.BridgeAddress, &s.c.Logger)
	if err != nil {
		s.c.Logger.Error().Err(err).Msg("cannot init bridge contract of sender chain.")
		return err
	}
	s.bridgeContract = c
	return nil
}

func (s *SenderChain) start(ch chan error) {
	ch <- s.pollBlocks()
}

func (s *SenderChain) pollBlocks() error {
	var currentBlock = s.startBlock
	log.Info().Msgf("Polling Blocks.. current block:%d", currentBlock.Int64())

	var retry = BlockRetryLimit
	for {
		// No more retries, goto next block
		if retry == 0 {
			log.Error().Msg("Polling failed, retries exceeded")
			s.c.EvmClient.Close()
			return nil
		}

		latestBlock, err := s.c.EvmClient.LatestBlock()
		if err != nil {
			log.Error().Any("block", currentBlock).Err(err).Msg("Unable to get latest block")
			retry--
			time.Sleep(BlockRetryInterval)
			continue
		}

		if big.NewInt(0).Sub(latestBlock, currentBlock).Cmp(s.blockConfirmations) == -1 {
			log.Debug().Any("target", currentBlock).Any("latest", latestBlock).Msg("Block not ready, will retry")
			time.Sleep(BlockRetryInterval)
			continue
		}

		err = s.getDepositEventsForBlock(currentBlock)
		if err != nil {
			s.c.Logger.Error().Err(err).Any("block", currentBlock).Msg("Failed to get events for block")
			retry--
			continue
		}

		// Write to block store. Not a critical operation, no need to retry
		err = s.blockStore.StoreBlock(currentBlock)
		if err != nil {
			s.c.Logger.Error().Err(err).Any("block", currentBlock).Msg("Failed to write latest block to blockstore")
		}

		// Goto next block and reset retry counter
		currentBlock.Add(currentBlock, big.NewInt(1))
		retry = BlockRetryLimit
	}
}

// getDepositEventsForBlock looks for the deposit event in the latest block
func (s *SenderChain) getDepositEventsForBlock(latestBlock *big.Int) error {
	s.c.Logger.Debug().Any("block", latestBlock).Msg("Querying block for deposit events")
	logs, err := s.c.EvmClient.FetchEventLogs(context.Background(), *s.bridgeContract.Contract.ContractAddress(), message.Deposit.String(), latestBlock, latestBlock)
	if err != nil {
		return fmt.Errorf("unable to Filter Logs: %w", err)
	}

	// read through the log events and handle their deposit event if handler is recognized
	for _, log := range logs {
		tx, pending, err := s.c.EvmClient.TransactionByHash(context.Background(), log.TxHash)
		if err != nil {
			return fmt.Errorf("error cannot get transaction by hash. hash:%s, err:%w", log.TxHash, err)
		}
		if !pending {
			signer := types.LatestSignerForChainID(tx.ChainId())
			sender, err := types.Sender(signer, tx)
			if err != nil {
				return fmt.Errorf("error cannot get sender from transaction. err:%w", err)
			}
			msg := message.NewDepositMessage(sender, tx.Value())
			s.msgChan <- msg
			s.c.Logger.Info().Msgf("sender chain send messge to receiver chain. sender:%s, value:%d", msg.Sender.Hex(), msg.Value.Int64())
		}
	}
	return nil
}
