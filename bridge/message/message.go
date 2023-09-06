package message

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type DepositMessage struct {
	Sender common.Address
	Value  *big.Int
}

func NewDepositMessage(sender common.Address, value *big.Int) DepositMessage {
	return DepositMessage{Sender: sender, Value: value}
}
