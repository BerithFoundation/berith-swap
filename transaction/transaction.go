package transaction

import (
	"berith-swap/transaction/signer"
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type CommonTransaction interface {
	Hash() common.Hash
	RawWithSignature(signer signer.Signer, domainID *big.Int) ([]byte, error)
}

type TX struct {
	tx *types.Transaction
}

func (a *TX) RawWithSignature(signer signer.Signer, domainID *big.Int) ([]byte, error) {
	opts, err := newTransactorWithChainID(signer, domainID)
	if err != nil {
		return nil, err
	}
	tx, err := opts.Signer(signer.CommonAddress(), a.tx)
	if err != nil {
		return nil, err
	}
	a.tx = tx

	data, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return data, nil
}

func NewTransaction(nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrices []*big.Int, data []byte) (CommonTransaction, error) {
	if len(gasPrices) > 1 {
		return newDynamicFeeTransaction(nonce, to, amount, gasLimit, gasPrices[0], gasPrices[1], data), nil
	} else {
		return newTransaction(nonce, to, amount, gasLimit, gasPrices[0], data), nil
	}
}

func newDynamicFeeTransaction(nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasTipCap *big.Int, gasFeeCap *big.Int, data []byte) *TX {
	tx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		To:        to,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Gas:       gasLimit,
		Value:     amount,
		Data:      data,
	})
	return &TX{tx: tx}
}

func newTransaction(nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *TX {
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       to,
		Value:    amount,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})
	return &TX{tx: tx}
}

func (a *TX) Hash() common.Hash {
	return a.tx.Hash()
}

func newTransactorWithChainID(s signer.Signer, chainID *big.Int) (*bind.TransactOpts, error) {
	keyAddr := s.CommonAddress()
	if chainID == nil {
		return nil, bind.ErrNoChainID
	}
	signer := types.LatestSignerForChainID(chainID)
	return &bind.TransactOpts{
		From: keyAddr,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != keyAddr {
				return nil, bind.ErrNotAuthorized
			}
			signature, err := s.Sign(signer.Hash(tx).Bytes())
			if err != nil {
				return nil, err
			}
			return tx.WithSignature(signer, signature)
		},
		Context: context.Background(),
	}, nil
}
