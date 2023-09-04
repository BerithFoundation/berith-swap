package contract

import (
	"berith-swap/evmgaspricer"
	"berith-swap/transaction"
	"berith-swap/transaction/signer"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

const (
	ERC20Address = "0x4C6c0101B74f1789409EAB5E1D542057512472bD"
)

var (
	gasPrice = big.NewInt(25000000000)
)

func InitializeClient(
	senderKeyPair *signer.Keypair,
) (*EvmClient, error) {
	ethClient, err := NewEvmClient(senderKeyPair)
	if err != nil {
		log.Error().Err(fmt.Errorf("eth client initialization error: %v", err))
		return nil, err
	}
	return ethClient, nil
}

func InitializeTransactor(
	gasPrice *big.Int,
	txFabric transaction.TxFabric,
	client *EvmClient,
) (transaction.Transactor, error) {
	var trans transaction.Transactor

	gasPricer := evmgaspricer.NewLondonGasPriceClient(
		client,
		&evmgaspricer.GasPricerOpts{UpperLimitFeePerGas: gasPrice},
	)
	trans = transaction.NewSignAndSendTransactor(txFabric, gasPricer, client)

	return trans, nil
}

func InitErc20Contract(privKey string) (*ERC20Contract, error) {

	sender, err := signer.NewKeypairFromPrivateKey(privKey)
	if err != nil {
		return nil, err
	}
	c, err := InitializeClient(sender)
	if err != nil {
		return nil, err
	}
	t, err := InitializeTransactor(gasPrice, transaction.NewTransaction, c)
	if err != nil {
		return nil, err
	}
	return NewERC20Contract(c, common.HexToAddress(ERC20Address), t), nil
}
