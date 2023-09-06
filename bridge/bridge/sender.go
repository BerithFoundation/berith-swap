package bridge

import (
	"berith-swap/bridge/blockstore"
	"berith-swap/bridge/chain"
	"berith-swap/bridge/config"
	"berith-swap/bridge/message"
	"berith-swap/bridge/util"
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
)

var (
	BlockRetryLimit           = 5
	BlockRetryInterval        = time.Second * 5
	DefaultBlockConfirmations = big.NewInt(10)
)

type SenderChain struct {
	c          *chain.Chain
	msgChan    chan<- message.DepositMessage
	blockStore *blockstore.Blockstore
	stop       chan struct{}
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
		logger.Error().Msgf("cannot get block confirmations from config. set default:%d", DefaultBlockConfirmations.Int64())
		blockConfirmations = DefaultBlockConfirmations
	}
	return &SenderChain{
		c:          chain,
		msgChan:    ch,
		blockStore: bs,
	}
}

func (s *SenderChain) pollBlocks() error {
	var currentBlock = s.c.StartBlock
	log.Info().Msgf("Polling Blocks...", "block", currentBlock)

	var retry = BlockRetryLimit
	for {
		select {
		case <-s.c.stop:
			return errors.New("polling terminated")
		default:
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

			// Sleep if the difference is less than BlockDelay; (latest - current) < BlockDelay
			if big.NewInt(0).Sub(latestBlock, currentBlock).Cmp(s.BlockConfirmations) == -1 {
				log.Debug().Any("target", currentBlock).Any("latest", latestBlock).Msg("Block not ready, will retry")
				time.Sleep(BlockRetryInterval)
				continue
			}

			// Coin 수신 체인 전용
			err = s.getDepositEventsForBlock(currentBlock)
			if err != nil {
				s.c.Logger.Error().Err(err).Any("block", currentBlock).Msg("Failed to get events for block")
				retry--
				continue
			}

			// Write to block store. Not a critical operation, no need to retry
			err = s.BlockStore.StoreBlock(currentBlock)
			if err != nil {
				s.c.Logger.Error().Err(err).Any("block", currentBlock).Msg("Failed to write latest block to blockstore")
			}

			// Goto next block and reset retry counter
			currentBlock.Add(currentBlock, big.NewInt(1))
			retry = BlockRetryLimit
		}
	}
}

// getDepositEventsForBlock looks for the deposit event in the latest block
func (s *SenderChain) getDepositEventsForBlock(latestBlock *big.Int) error {
	s.c.Logger.Debug().Any("block", latestBlock).Msg("Querying block for deposit events")
	query := s.c.EvmClient.buildQuery(s.EvmClient.Keypair().CommonAddress(), latestBlock, latestBlock)

	// FetchEventLogs
	logs, err := s.c.EvmClient.FilterLogs(context.Background(), query)
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
		}
	}

	return nil
}
