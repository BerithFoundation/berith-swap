package bridge

import (
	"berith-swap/bridge/blockstore"
	"berith-swap/bridge/chain"
	"berith-swap/bridge/config"
	"berith-swap/bridge/contract"
	"berith-swap/bridge/message"
	"berith-swap/bridge/store"
	"berith-swap/bridge/store/mariadb"
	"berith-swap/bridge/transaction"
	"berith-swap/bridge/util"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// ReceiverChain은 SenderChain 측에서 전송된 코인 예치 메시지를 수신하고 토큰 컨트랙트를 통해 해당 사용자에게 토큰을 전송합니다.
type ReceiverChain struct {
	c             *chain.Chain
	msgChan       <-chan message.DepositMessage
	erc20Contract *contract.ERC20Contract
	blockStore    *blockstore.Blockstore
	stop          chan struct{}
	store         *store.Store
}

// NewReceiverChain는 ReceiverChain을 생성합니다.
func NewReceiverChain(ch <-chan message.DepositMessage, cfg *config.Config, idx int, bs *blockstore.Blockstore) *ReceiverChain {
	chainCfg := cfg.ChainConfig[idx]

	chain, err := chain.NewChain(cfg, idx)
	if err != nil {
		chain.Logger.Panic().Err(err).Msgf("cannot init chain. idx:%d", idx)
	}

	newErc20, err := contract.InitErc20Contract(chain.EvmClient, chainCfg.Erc20Address, &chain.Logger)
	if err != nil {
		chain.Logger.Panic().Err(err).Msg("cannot init erc20 contract")
	}

	store, err := store.NewStore(cfg.DBSource)
	if err != nil {
		chain.Logger.Panic().Err(err).Msg("cannot init remote db store")
	}

	rc := ReceiverChain{
		c:             chain,
		blockStore:    bs,
		msgChan:       ch,
		erc20Contract: newErc20,
		stop:          make(chan struct{}),
		store:         store,
	}
	rc.setReceiverErc20Contract(chainCfg)
	go rc.listen()
	return &rc
}

// setReceiverErc20Contract는 receiver chain의 erc20 contract를 설정합니다.
func (s *ReceiverChain) setReceiverErc20Contract(chainCfg *config.RawChainConfig) error {
	if chainCfg.Erc20Address == "" {
		err := fmt.Errorf("receiver chain dosen't have erc20 contract address. Chain:%s, Idx:%d", s.c.Name, chainCfg.Idx)
		s.c.Logger.Error().Err(err).Msg("check config.json")
		return err
	}

	err := s.c.EvmClient.EnsureHasBytecode(common.HexToAddress(chainCfg.Erc20Address))
	if err != nil {
		s.c.Logger.Panic().Err(err).Msgf("contract dosen't exist this chain url:%s", s.c.Endpoint)
	}

	c, err := contract.InitErc20Contract(s.c.EvmClient, chainCfg.Erc20Address, &s.c.Logger)
	if err != nil {
		s.c.Logger.Error().Err(err).Msg("cannot init erc20 contract of sender chain.")
		return err
	}
	s.erc20Contract = c
	return nil
}

// start는 ReceiverChain을 시작합니다.
func (r *ReceiverChain) start(ch chan error) {
	ch <- r.listen()
}

// listen은 ReceiverChain의 메시지 채널을 수신하고 토큰을 전송합니다.
func (r *ReceiverChain) listen() error {
	for {
		select {
		case m := <-r.msgChan:
			valErr := util.ValidateStruct(m)
			if valErr != nil {
				r.c.Logger.Warn().Msgf("Invalid deposit message. %s", valErr.Error())
				continue
			}
			err := r.SendToken(m)
			if err != nil {
				r.c.Logger.Error().Err(err).Msg("error occured during send token. stop receiver chain.")
				return err
			}
		case <-r.stop:
			r.c.Logger.Error().Msg("receiver chain got stop sign")
			return errors.New("receiver chain stopped")
		}

	}
}

// SendToken은 Deposit 메시지를 수신하고 토큰을 전송합니다.
func (r *ReceiverChain) SendToken(m message.DepositMessage) error {
	history, err := r.store.GetBersSwapHistory(context.Background(), m.SenderTxHash)
	if err != nil {
		if err != sql.ErrNoRows {
			r.c.Logger.Error().Err(err).Msgf("cannot get swab history from remote store. hash:%s", m.SenderTxHash)
			return err
		}
	}

	if history.SenderTxHash != "" {
		r.c.Logger.Warn().Msgf("swap already excecuted, ignore deposit message. sender tx: %s", history.SenderTxHash)
		return nil
	}

	txHash, err := r.erc20Contract.Transfer(m.Receiver, m.Amount, transaction.TransactOptions{GasLimit: r.c.GasLimit.Uint64()})
	if err != nil {
		r.c.Logger.Error().Err(err).Any("Address", m.Receiver.Hex()).Any("Value", m.Amount.Uint64()).Msg("transaction submit failed.")
		return err
	}

	rec, err := r.erc20Contract.WaitAndReturnTxReceipt(txHash)
	if err != nil {
		r.c.Logger.Error().Err(err).Msgf("cannot get tx receipt hash:%s", txHash.Hex())
		return err
	}

	gasUsed := new(big.Float).Quo(new(big.Float).SetInt(new(big.Int).SetUint64(rec.GasUsed)), new(big.Float).SetInt(big.NewInt(1e18)))
	r.c.Logger.Info().Msgf("receive tx receipt successfully. Block: %s, Tx Hash: %s, GasUsed: %s", rec.BlockNumber, txHash.Hex(), gasUsed.String())

	err = r.blockStore.StoreBlock(new(big.Int).SetUint64(m.BlockNumber))
	if err != nil {
		r.c.Logger.Error().Err(err).Msg("Failed to write latest block to blockstore")
		return err
	}
	r.c.Logger.Info().Msgf("saved the block number where the deposit event occurred. number: %d", m.BlockNumber)

	err = r.store.CreateSwapHistoryTx(context.Background(), mariadb.CreateBersSwapHistoryParams{
		SenderTxHash:   m.SenderTxHash,
		ReceiverTxHash: txHash.Hex(),
		Amount:         m.Amount.Int64(),
	})
	if err != nil {
		r.c.Logger.Error().Err(err).Msg("Failed to store swap history to remote db")
		return err
	}

	r.c.Logger.Info().Msgf("saved sender tx hash into remote db store. sender Tx: %s", m.SenderTxHash)
	return nil
}

// Stop는 ReceiverChain을 종료합니다.
func (r *ReceiverChain) Stop() {
	r.store.Stop()
	r.stop <- struct{}{}
}
