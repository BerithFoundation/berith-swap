package message

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type EventSig string

func (e *EventSig) String() string {
	return string(*e)
}

func (es EventSig) GetTopic() common.Hash {
	return crypto.Keccak256Hash([]byte(es))
}

var (
	Deposit EventSig = "Deposit(uint64,address,address,uint256)"
)
