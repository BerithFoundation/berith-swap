package contract

import (
	"berith-swap/bridge/connection"
	"berith-swap/bridge/evmgaspricer"
	"berith-swap/bridge/transaction"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
)

var (
	KlaytnBaseFee = big.NewInt(25000000000) // 25 Gwei
	BerithBaseFee = big.NewInt(0)
)

// InitializeTransactor는 gas price clinet와 함께 Transactor를 초기화한다.
// baseFee + tip으로 지불할 총 gas fee의 제한을 GasPricerOpts에 저공하여 설정한다.
func InitializeTransactor(
	limitGP *big.Int,
	txFabric transaction.TxFabric,
	client *connection.EvmClient,
) (transaction.Transactor, error) {
	var trans transaction.Transactor

	gasPricer := evmgaspricer.NewLondonGasPriceClient(
		client,
		&evmgaspricer.GasPricerOpts{UpperLimitFeePerGas: limitGP},
	)
	trans = transaction.NewSignAndSendTransactor(txFabric, gasPricer, client)

	return trans, nil
}

func InitErc20Contract(c *connection.EvmClient, erc20Addr string, logger *zerolog.Logger) (*ERC20Contract, error) {

	t, err := InitializeTransactor(KlaytnBaseFee, transaction.NewTransaction, c)
	if err != nil {
		return nil, err
	}
	return NewERC20Contract(c, common.HexToAddress(erc20Addr), t, logger), nil
}

func IniBridgeContract(c *connection.EvmClient, bridgeAddr string, logger *zerolog.Logger) (*BridgeContract, error) {

	t, err := InitializeTransactor(BerithBaseFee, transaction.NewTransaction, c)
	if err != nil {
		return nil, err
	}
	return NewBridgeContract(c, common.HexToAddress(bridgeAddr), t, logger), nil
}
