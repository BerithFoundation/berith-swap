package bridge

import (
	chain "berith-swap/bridge/chain"
	contract "berith-swap/bridge/contract"
	message "berith-swap/bridge/message"
	transaction "berith-swap/bridge/transaction"

	"github.com/ethereum/go-ethereum/common"
)

// ReceiverChain은 SenderChain 측에서 전송된 코인 예치 메시지를 수신하고 토큰 컨트랙트를 통해 해당 사용자에게 토큰을 전송합니다.
type ReceiverChain struct {
	chain   *chain.Chain
	msgChan <-chan message.DepositMessage
	erc20   *contract.ERC20Contract
	stop    chan struct{}
}

func NewReceiverChain(chain *chain.Chain, c <-chan message.DepositMessage, contractAddr string) *ReceiverChain {
	newErc20, err := contract.InitErc20Contract(chain.EvmClient, contractAddr, &chain.Logger)
	if err != nil {
		chain.Logger.Panic().Err(err).Msg("cannot init erc20 contract")
	}

	err = chain.EvmClient.EnsureHasBytecode(common.HexToAddress(contractAddr))
	if err != nil {
		chain.Logger.Panic().Err(err).Msgf("contract dosen't exist this chain url:%s", chain.Endpoint)
	}

	rc := ReceiverChain{
		chain:   chain,
		msgChan: c,
		erc20:   newErc20,
	}
	go rc.loop()
	return &rc
}

func (r *ReceiverChain) loop() {
	for {
		m := <-r.msgChan
		err := r.SendToken(m)
		if err != nil {
			r.chain.Logger.Error().Err(err).Msg("error occured during send token.")
		}
	}
}

func (r *ReceiverChain) SendToken(m message.DepositMessage) error {

	txHash, err := r.erc20.Transfer(m.Sender, m.Value, transaction.TransactOptions{GasLimit: r.chain.GasLimit.Uint64()})
	if err != nil {
		return err
	}
	r.chain.Logger.Info().Msgf("Transaction submit. Tx Hash : %s", txHash.Hex())
	return nil
}
