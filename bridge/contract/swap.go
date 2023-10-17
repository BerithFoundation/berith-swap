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

type SwapContract struct {
	Contract
	Logger *zerolog.Logger
}

func NewSwapContract(
	client transaction.ContractCallerDispatcher,
	BridgeContractAddress common.Address,
	transactor transaction.Transactor,
	logger *zerolog.Logger,
) *SwapContract {
	a, _ := abi.JSON(strings.NewReader(consts.BerithSwapABI))
	b := common.FromHex(consts.BerithSwapBin)
	return &SwapContract{
		Contract: NewContract(BridgeContractAddress, a, b, client, transactor, logger),
		Logger:   logger,
	}
}

func (b *SwapContract) WaitAndReturnTxReceipt(hash *common.Hash) (*types.Receipt, error) {
	return b.Contract.client.WaitAndReturnTxReceipt(*hash)
}

func (b *SwapContract) Deposit(receiver common.Address,
	opts transaction.TransactOptions) (*common.Hash, error) {
	valueFloat, _ := new(big.Float).SetString(opts.Value.String())
	b.Logger.Debug().Msgf("deposit %s BERS, receiver:%s", new(big.Float).Quo(valueFloat, new(big.Float).SetUint64(1e18)).String(), receiver.Hex())
	return b.ExecuteTransaction("deposit", opts, receiver)
}

func (b *SwapContract) GetBalance(address common.Address) (*big.Int, error) {
	b.Logger.Debug().Msgf("Getting balance for %s", address.String())
	res, err := b.CallContract("balanceOf", address)
	if err != nil {
		return nil, err
	}
	abi := abi.ConvertType(res[0], new(big.Int)).(*big.Int)
	return abi, nil
}

func (b *SwapContract) TransferFunds(
	opts transaction.TransactOptions,
) (*common.Hash, error) {
	b.Logger.Debug().Msg("withdraw all bers from swap contract")
	return b.ExecuteTransaction("transferFunds", opts)
}
