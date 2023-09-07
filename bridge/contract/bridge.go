package contract

import (
	"berith-swap/bridge/contract/consts"
	"berith-swap/bridge/transaction"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog"
)

type BridgeContract struct {
	Contract
	Logger *zerolog.Logger
}

func NewBridgeContract(
	client transaction.ContractCallerDispatcher,
	BridgeContractAddress common.Address,
	transactor transaction.Transactor,
	logger *zerolog.Logger,
) *BridgeContract {
	a, _ := abi.JSON(strings.NewReader(consts.BerithSwapABI))
	b := common.FromHex(consts.BerithSwapBin)
	return &BridgeContract{
		Contract: NewContract(BridgeContractAddress, a, b, client, transactor),
		Logger:   logger,
	}
}

func (b *BridgeContract) WaitAndReturnTxReceipt(hash *common.Hash) (*types.Receipt, error) {
	return b.Contract.client.WaitAndReturnTxReceipt(*hash)
}

func (b *BridgeContract) GetBalance(address common.Address) (*big.Int, error) {
	b.Logger.Debug().Msgf("Getting balance for %s", address.String())
	res, err := b.CallContract("balanceOf", address)
	if err != nil {
		return nil, err
	}
	abi := abi.ConvertType(res[0], new(big.Int)).(*big.Int)
	return abi, nil
}

func (b *BridgeContract) TransferFunds(
	to common.Address,
	amount *big.Int,
	opts transaction.TransactOptions,
) (*common.Hash, error) {
	b.Logger.Debug().Msgf("transfer %s tokens to %s", amount.String(), to.String())
	return b.ExecuteTransaction("transferFunds", opts, to, amount)
}
