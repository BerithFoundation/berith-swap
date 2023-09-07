package bridge

import (
	"berith-swap/bridge/blockstore"
	chain "berith-swap/bridge/chain"
	"berith-swap/bridge/config"
	contract "berith-swap/bridge/contract"
	message "berith-swap/bridge/message"
	transaction "berith-swap/bridge/transaction"
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
}

func NewReceiverChain(ch <-chan message.DepositMessage, cfg *config.Config, idx int) *ReceiverChain {
	chainCfg := cfg.ChainConfig[idx]

	chain, err := chain.NewChain(cfg, idx)
	if err != nil {
		chain.Logger.Panic().Err(err).Msgf("cannot init chain. idx:%d", idx)
	}
	bs, err := blockstore.NewBlockstore(cfg.BlockStorePath, chain.Name)
	if err != nil {
		chain.Logger.Panic().Err(err).Msg("cannot init block store")
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
		m := <-r.msgChan
		err := r.SendToken(m)
		if err != nil {
			r.c.Logger.Error().Err(err).Msg("error occured during send token. stop receiver chain.")
			return err
		}
	}
}

func (r *ReceiverChain) SendToken(m message.DepositMessage) error {
	txHash, err := r.erc20Contract.Transfer(m.Sender, m.Value, transaction.TransactOptions{GasLimit: r.c.GasLimit.Uint64()})
	if err != nil {
		r.c.Logger.Error().Err(err).Msgf("transaction submit failed.", txHash.Hex())
		return err
	}

	rec, err := r.erc20Contract.WaitAndReturnTxReceipt(txHash)
	if err != nil {
		r.c.Logger.Error().Err(err).Msgf("cannot get tx receipt hash:%s", txHash.Hex())
		return err
	}

	gasUsed := new(big.Float).Quo(new(big.Float).SetInt(new(big.Int).SetUint64(rec.GasUsed)), new(big.Float).SetInt(big.NewInt(1e18)))
	r.c.Logger.Info().Msgf("receive tx receipt successfully. Tx Hash : %s, GasUsed: %s", txHash.Hex(), rec.BlockNumber, gasUsed.String())
	return nil
}
