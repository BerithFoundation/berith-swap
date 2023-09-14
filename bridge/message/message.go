package message

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type DepositMessage struct {
	BlockNumber  uint64         `validate:"required"`
	Receiver     common.Address `validate:"required"`
	Amount       *big.Int       `validate:"required"`
	SenderTxHash string         `validate:"required,len=66"`
}

func NewDepositMessage(blockNumber uint64, receiver common.Address, amount *big.Int, hash string) DepositMessage {
	return DepositMessage{BlockNumber: blockNumber, Receiver: receiver, Amount: amount, SenderTxHash: hash}
}
