package bridge

import (
	"berith-swap/bridge/blockstore"
	"berith-swap/bridge/config"
	"berith-swap/bridge/message"

	"github.com/rs/zerolog/log"
)

const (
	MsgChanSize = 50
	SenderIdx   = 0
	ReceiverIdx = 1
)

// Bridge는 SenderChain과 ReceiverChain을 생성하고 시작합니다.
type Bridge struct {
	sc *SenderChain
	rc *ReceiverChain
}

// NewBridge는 SenderChain과 ReceiverChain을 생성합니다.
func NewBridge(cfg *config.Config) *Bridge {
	br := new(Bridge)

	msgChan := make(chan message.DepositMessage)
	bs, err := blockstore.NewBlockstore(cfg.BlockStorePath, cfg.ChainConfig[0].Name)
	if err != nil {
		log.Error().Err(err).Msg("cannot initialize block store")
	}
	sc := NewSenderChain(msgChan, cfg, SenderIdx, bs)
	rc := NewReceiverChain(msgChan, cfg, ReceiverIdx, bs)

	br.sc = sc
	br.rc = rc
	return br
}

// Start는 SenderChain과 ReceiverChain을 시작합니다.
func (b *Bridge) Start() error {
	ch := make(chan error)
	go b.sc.start(ch)
	go b.rc.start(ch)

	return <-ch
}

// Stop는 SenderChain과 ReceiverChain을 종료합니다.
func (b *Bridge) Stop() {
	b.sc.Stop()
	b.rc.Stop()
}
