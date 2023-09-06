package transaction

import (
	"context"
	"math/big"

	"dario.cat/mergo"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

const DefaultGasLimit = 2000000

var DefaultTransactionOptions = TransactOptions{
	GasLimit: DefaultGasLimit,
	GasPrice: big.NewInt(0),
	Value:    big.NewInt(0),
}

type TransactOptions struct {
	GasLimit uint64
	GasPrice *big.Int
	Value    *big.Int
	Nonce    *big.Int
	ChainID  *big.Int
	Priority uint8
}

var TxPriorities = map[string]uint8{
	"none":   0,
	"slow":   1,
	"medium": 2,
	"fast":   3,
}

func MergeTransactionOptions(primary *TransactOptions, additional *TransactOptions) error {
	if err := mergo.Merge(primary, additional); err != nil {
		return err
	}

	return nil
}

type Transactor interface {
	Transact(to *common.Address, data []byte, opts TransactOptions) (*common.Hash, error)
}

type signAndSendTransactor struct {
	TxFabric       TxFabric
	gasPriceClient GasPricer
	client         ClientDispatcher
}

func NewSignAndSendTransactor(txFabric TxFabric, gasPriceClient GasPricer, client ClientDispatcher) Transactor {
	return &signAndSendTransactor{
		TxFabric:       txFabric,
		gasPriceClient: gasPriceClient,
		client:         client,
	}
}

func (t *signAndSendTransactor) Transact(to *common.Address, data []byte, opts TransactOptions) (*common.Hash, error) {
	t.client.LockNonce()
	n, err := t.client.UnsafeNonce()
	if err != nil {
		t.client.UnlockNonce()
		return &common.Hash{}, err
	}

	err = MergeTransactionOptions(&opts, &DefaultTransactionOptions)
	if err != nil {
		t.client.UnlockNonce()
		return &common.Hash{}, err
	}

	gp := []*big.Int{opts.GasPrice}
	if opts.GasPrice.Cmp(big.NewInt(0)) == 0 {
		gp, err = t.gasPriceClient.GasPrice(&opts.Priority)
		if err != nil {
			t.client.UnlockNonce()
			return &common.Hash{}, err
		}
	}

	tx, err := t.TxFabric(n.Uint64(), to, opts.Value, opts.GasLimit, gp, data)
	if err != nil {
		t.client.UnlockNonce()
		return &common.Hash{}, err
	}

	h, err := t.client.SignAndSendTransaction(context.TODO(), tx)
	if err != nil {
		t.client.UnlockNonce()
		log.Error().Err(err)
		return &common.Hash{}, err
	}

	err = t.client.UnsafeIncreaseNonce()
	t.client.UnlockNonce()
	if err != nil {
		return &common.Hash{}, err
	}

	_, err = t.client.WaitAndReturnTxReceipt(h)
	if err != nil {
		return &common.Hash{}, err
	}

	return &h, nil
}
