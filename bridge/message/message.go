package message

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type DepositMessage struct {
	BlockNumber uint64
	Receiver    common.Address
	Value       *big.Int
}

func NewDepositMessage(blockNumber uint64, receiver common.Address, value *big.Int) DepositMessage {
	return DepositMessage{BlockNumber: blockNumber, Receiver: receiver, Value: value}
}
