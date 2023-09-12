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
	DefaultGasPrice = big.NewInt(25000000000)
)

func InitializeTransactor(
	gasPrice *big.Int,
	txFabric transaction.TxFabric,
	client *connection.EvmClient,
) (transaction.Transactor, error) {
	var trans transaction.Transactor

	gasPricer := evmgaspricer.NewLondonGasPriceClient(
		client,
		&evmgaspricer.GasPricerOpts{UpperLimitFeePerGas: gasPrice},
	)
	trans = transaction.NewSignAndSendTransactor(txFabric, gasPricer, client)

	return trans, nil
}

func InitErc20Contract(c *connection.EvmClient, erc20Addr string, logger *zerolog.Logger) (*ERC20Contract, error) {

	t, err := InitializeTransactor(DefaultGasPrice, transaction.NewTransaction, c)
	if err != nil {
		return nil, err
	}
	return NewERC20Contract(c, common.HexToAddress(erc20Addr), t, logger), nil
}

func IniBridgeContract(c *connection.EvmClient, bridgeAddr string, logger *zerolog.Logger) (*BridgeContract, error) {

	t, err := InitializeTransactor(DefaultGasPrice, transaction.NewTransaction, c)
	if err != nil {
		return nil, err
	}
	return NewBridgeContract(c, common.HexToAddress(bridgeAddr), t, logger), nil
}
