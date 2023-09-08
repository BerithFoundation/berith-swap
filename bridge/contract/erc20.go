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

type ERC20Contract struct {
	Contract
	Logger *zerolog.Logger
}

func NewERC20Contract(
	client transaction.ContractCallerDispatcher,
	erc20ContractAddress common.Address,
	transactor transaction.Transactor,
	logger *zerolog.Logger,
) *ERC20Contract {
	a, _ := abi.JSON(strings.NewReader(consts.BersTokenABI))
	b := common.FromHex(consts.BersTokenBin)
	return &ERC20Contract{
		Contract: NewContract(erc20ContractAddress, a, b, client, transactor),
		Logger:   logger,
	}
}

func (c *ERC20Contract) WaitAndReturnTxReceipt(hash *common.Hash) (*types.Receipt, error) {
	return c.Contract.client.WaitAndReturnTxReceipt(*hash)
}

func (c *ERC20Contract) GetBalance(address common.Address) (*big.Int, error) {
	c.Logger.Debug().Msgf("Getting balance for %s", address.String())
	res, err := c.CallContract("balanceOf", address)
	if err != nil {
		return nil, err
	}
	b := abi.ConvertType(res[0], new(big.Int)).(*big.Int)
	return b, nil
}

func (c *ERC20Contract) Transfer(
	to common.Address,
	amount *big.Int,
	opts transaction.TransactOptions,
) (*common.Hash, error) {
	c.Logger.Debug().Msgf("transfer %s tokens to %s", amount.String(), to.String())
	return c.ExecuteTransaction("transfer", opts, to, amount)
}
