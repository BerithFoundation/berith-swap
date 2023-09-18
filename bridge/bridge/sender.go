package bridge

import (
	"berith-swap/bridge/blockstore"
	"berith-swap/bridge/chain"
	"berith-swap/bridge/config"
	"berith-swap/bridge/contract"
	"berith-swap/bridge/message"
	"berith-swap/bridge/util"
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

var (
	BlockRetryLimit           = 5
	BlockRetryInterval        = time.Second * 5
	DefaultBlockConfirmations = big.NewInt(10)
)

// SenderChain은 Bridge 컨트랙트를 모니터링하며 Deposit 이벤트가 감지되면 ReceiverChain으로 토큰 전송 메시지를 보냅니다.
type SenderChain struct {
	c                  *chain.Chain
	msgChan            chan<- message.DepositMessage
	blockStore         *blockstore.Blockstore
	blockConfirmations *big.Int
	bridgeContract     *contract.BridgeContract
	startBlock         *big.Int
	stop               chan struct{}
}

// NewSenderChain는 SenderChain을 생성합니다.
func NewSenderChain(ch chan<- message.DepositMessage, cfg *config.Config, idx int, bs *blockstore.Blockstore) *SenderChain {
	chain, err := chain.NewChain(cfg, idx)
	if err != nil {
		chain.Logger.Panic().Err(err).Msgf("cannot init chain. idx:%d", idx)
	}

	blockConfirmations, err := util.StringToBig(cfg.ChainConfig[idx].BlockConfirmations, 10) // contract 등록 유무로 isSender
	if err != nil {
		chain.Logger.Error().Msgf("cannot get block confirmations from config. set default:%d", DefaultBlockConfirmations.Int64())
		blockConfirmations = DefaultBlockConfirmations
	}

	startBlock, err := chain.EvmClient.LatestBlock()
	if err != nil {
		chain.Logger.Error().Err(err).Msgf("cannot get latest block through evmclient.")
		return nil
	}

	if cfg.IsLoaded {
		chain.Logger.Info().Msgf("try load latest block from block store isLoaded:%v", cfg.IsLoaded)
		startBlock, err = bs.TryLoadLatestBlock()
		if err != nil {
			chain.Logger.Panic().Err(err).Msgf("cannot load latest block from block store.")
		}
		chain.Logger.Info().Msgf("loaded latest block number form blockstore. path:%s, number:%d", bs.FullPath(), startBlock.Uint64())
	}

	chain.Logger.Info().Msgf("Latest block : %d", startBlock.Uint64())

	sc := SenderChain{
		c:                  chain,
		msgChan:            ch,
		blockStore:         bs,
		blockConfirmations: blockConfirmations,
		startBlock:         startBlock,
		stop:               make(chan struct{}),
	}

	sc.setSenderBridgeContract(cfg.ChainConfig[idx])
	return &sc
}

// setSenderBridgeContract는 SenderChain의 BridgeContract를 설정합니다.
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

// start는 SenderChain을 시작합니다.
func (s *SenderChain) start(ch chan error) {
	ch <- s.pollBlocks()

}

// pollBlocks는 SenderChain의 블록을 폴링하며 Deposit 이벤트를 감지합니다.
func (s *SenderChain) pollBlocks() error {
	var currentBlock = s.startBlock
	s.c.Logger.Info().Msgf("Polling Blocks.. current block:%s", currentBlock.String())

	var isStopped bool
	go func() {
		<-s.stop
		isStopped = true
	}()

	var retry = BlockRetryLimit
	for {
		if isStopped {
			s.c.Logger.Error().Msg("sender chain got stop sign")
			return errors.New("sender chain stopped")
		}
		// No more retries, goto next block
		if retry == 0 {
			err := fmt.Errorf("sender chain failed to poll block, retries exceeded")
			s.c.Logger.Error().Err(err).Msg("")
			s.c.EvmClient.Close()
			return err
		}

		latestBlock, err := s.c.EvmClient.LatestBlock()
		if err != nil {
			s.c.Logger.Error().Any("block", currentBlock.String()).Err(err).Msg("Unable to get latest block")
			retry--
			time.Sleep(BlockRetryInterval)
			continue
		}

		if big.NewInt(0).Sub(latestBlock, currentBlock).Cmp(s.blockConfirmations) == -1 {
			s.c.Logger.Debug().Any("current", currentBlock.String()).Any("latest", latestBlock.String()).Msg("Block not ready, will retry")
			time.Sleep(BlockRetryInterval)
			continue
		}

		msgs, err := s.getDepositEventsForBlock(currentBlock)
		if err != nil {
			s.c.Logger.Error().Err(err).Any("block", currentBlock).Msg("Failed to get events for block")
			retry--
			continue
		}

		s.SendMsgs(msgs)
		if len(msgs) > 0 {
			s.c.Logger.Info().Msgf("message sended compeletely, msgs:%d", len(msgs))
		}

		// Goto next block and reset retry counter
		currentBlock.Add(currentBlock, big.NewInt(1))
		retry = BlockRetryLimit
	}
}

// getDepositEventsForBlock는 블록에서 Deposit 이벤트를 탐색합니다.
func (s *SenderChain) getDepositEventsForBlock(latestBlock *big.Int) ([]message.DepositMessage, error) {
	s.c.Logger.Debug().Any("block", latestBlock.String()).Msg("Querying block for deposit events")
	logs, err := s.c.EvmClient.FetchEventLogs(context.Background(), *s.bridgeContract.Contract.ContractAddress(), message.Deposit, latestBlock, latestBlock)
	if err != nil {
		return nil, fmt.Errorf("unable to Filter Logs: %w", err)
	}

	msgs := []message.DepositMessage{}
	// read through the log events and handle their deposit event if handler is recognized
	for _, log := range logs {
		tx, pending, err := s.c.EvmClient.TransactionByHash(context.Background(), log.TxHash)
		if err != nil {
			return nil, fmt.Errorf("error cannot get transaction by hash. hash:%s, err:%w", log.TxHash, err)
		}
		if !pending {
			receiver := common.BytesToAddress(log.Topics[1].Bytes())
			msg := message.NewDepositMessage(log.BlockNumber, receiver, big.NewInt(0).Div(tx.Value(), big.NewInt(1e18)), log.TxHash.Hex())
			msgs = append(msgs, msg)
		}
	}
	return msgs, nil
}

// SendMsgs는 ReceiverChain으로 메시지를 전송합니다.
func (s *SenderChain) SendMsgs(msgs []message.DepositMessage) {
	for _, msg := range msgs {
		s.msgChan <- msg
		s.c.Logger.Info().Msgf("sender chain send messge to receiver chain. block:%d receiver:%s, value:%s", msg.BlockNumber, msg.Receiver.Hex(), msg.Amount.String())
	}
}

// Stop는 SenderChain을 종료합니다.
func (s *SenderChain) Stop() {
	s.stop <- struct{}{}
}
