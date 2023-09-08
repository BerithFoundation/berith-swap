package bridge

import (
	"berith-swap/bridge/config"
	"berith-swap/bridge/message"
	"math/big"
)

const (
	MsgChanSize = 50
	SenderIdx   = 0
	ReceiverIdx = 1
)

type Bridge struct {
	sc *SenderChain
	rc *ReceiverChain
}

func NewBridge(cfg *config.Config) *Bridge {
	br := new(Bridge)

	msgChan := make(chan message.DepositMessage)
	sc := NewSenderChain(msgChan, cfg, SenderIdx)
	rc := NewReceiverChain(msgChan, cfg, ReceiverIdx)

	br.sc = sc
	br.rc = rc
	return br
}

func (b *Bridge) Start() error {
	ch := make(chan error)
	go b.sc.start(ch)
	go b.rc.start(ch)

	err := <-ch
	latest, bsErr := b.sc.blockStore.TryLoadLatestBlock()
	if err != nil {
		b.sc.c.Logger.Error().Err(bsErr).Msg("cannot load latest block number from blockstore after error occured.")
	}
	bsErr = b.sc.blockStore.StoreBlock(new(big.Int).Sub(latest, big.NewInt(1)))
	if err != nil {
		b.sc.c.Logger.Error().Err(bsErr).Msg("cannot store previous block number into blockstore after error occured.")
	}
	return err
}
