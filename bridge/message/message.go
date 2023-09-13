package message

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type DepositMessage struct {
	BlockNumber  uint64
	Receiver     common.Address
	Amount       *big.Int
	SenderTxHash string
}

func NewDepositMessage(blockNumber uint64, receiver common.Address, amount *big.Int, hash string) DepositMessage {
	return DepositMessage{BlockNumber: blockNumber, Receiver: receiver, Amount: amount, SenderTxHash: hash}
}
