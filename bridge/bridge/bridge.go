package bridge

import (
	"berith-swap/bridge/config"
	"berith-swap/bridge/message"
)

const (
	MsgChanSize = 10
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
	b.sc.start(ch)
	b.rc.start(ch)

	return <-ch
}
