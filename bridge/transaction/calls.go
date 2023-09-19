package transaction

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type ContractChecker interface {
	CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error)
}

type ContractCaller interface {
	CallContract(ctx context.Context, callArgs map[string]interface{}, blockNumber *big.Int) ([]byte, error)
}

type GasPricer interface {
	// make priority a pointer to uint8 to pass nil into all GasPrice functions (instead of magic numbers)
	GasPrice(priority *uint8) ([]*big.Int, error)
}

type ClientDispatcher interface {
	WaitAndReturnTxReceipt(h common.Hash) (*types.Receipt, error)
	SignAndSendTransaction(ctx context.Context, tx CommonTransaction) (common.Hash, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	GetTransactionByHash(h common.Hash) (tx *types.Transaction, isPending bool, err error)
	UnsafeNonce() (*big.Int, error)
	LockNonce()
	UnlockNonce()
	UnsafeIncreaseNonce() error
	From() common.Address
	LatestBlockNumber() (*big.Int, error)
	LatestBlock() (*types.Block, error)
	SuggestGasPrice(context.Context) (*big.Int, error)
}

type ContractCallerDispatcher interface {
	ContractCaller
	ClientDispatcher
	ContractChecker
}

func ToCallArg(msg ethereum.CallMsg) map[string]interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["data"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	return arg
}
