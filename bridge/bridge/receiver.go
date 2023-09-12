package bridge

import (
	"berith-swap/bridge/blockstore"
	chain "berith-swap/bridge/chain"
	"berith-swap/bridge/config"
	contract "berith-swap/bridge/contract"
	message "berith-swap/bridge/message"
	transaction "berith-swap/bridge/transaction"
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
}

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

	rc := ReceiverChain{
		c:             chain,
		blockStore:    bs,
		msgChan:       ch,
		erc20Contract: newErc20,
		stop:          make(chan struct{}),
	}
	rc.setReceiverErc20Contract(&chainCfg)
	go rc.listen()
	return &rc
}

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

func (r *ReceiverChain) start(ch chan error) {
	ch <- r.listen()
}

func (r *ReceiverChain) listen() error {
	for {
		select {
		case m := <-r.msgChan:
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

func (r *ReceiverChain) SendToken(m message.DepositMessage) error {
	txHash, err := r.erc20Contract.Transfer(m.Receiver, m.Value, transaction.TransactOptions{GasLimit: r.c.GasLimit.Uint64()})
	if err != nil {
		r.c.Logger.Error().Err(err).Any("Address", m.Receiver.Hex()).Any("Value", m.Value.Uint64()).Msg("transaction submit failed.")
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
	return nil
}

func (r *ReceiverChain) Stop() {
	r.stop <- struct{}{}
}
