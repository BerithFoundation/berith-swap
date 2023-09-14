package bridge

import (
	"berith-swap/bridge/blockstore"
	"berith-swap/bridge/message"
	"berith-swap/bridge/store/mariadb"
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestStoreHistory(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ch := make(chan message.DepositMessage)

	cfg := initTestconfig(t)

	chainCfg := cfg.ChainConfig[ReceiverIdx]

	bs, err := blockstore.NewBlockstore("../../run_test/blockstore", chainCfg.Name)
	require.NoError(t, err)
	require.NotNil(t, bs)

	rc := NewReceiverChain(ch, cfg, ReceiverIdx, bs)
	defer rc.Stop()

	txBytes := make([]byte, common.HashLength)
	_, err = rand.Read(txBytes)
	require.NoError(t, err)

	senderTx := hexutil.Encode(txBytes)
	err = rc.store.CreateSwapHistoryTx(context.Background(), mariadb.CreateBersSwapHistoryParams{
		SenderTxHash:   senderTx,
		ReceiverTxHash: "receiverTx",
		Amount:         1,
	})

	require.NoError(t, err)

	hist, err := rc.store.GetBersSwapHistory(context.Background(), senderTx)
	require.NoError(t, err)
	require.Equal(t, hist.Amount, int64(1))
	require.Equal(t, hist.ReceiverTxHash, "receiverTx")
	require.Equal(t, hist.SenderTxHash, senderTx)
	require.WithinDuration(t, hist.CreatedAt.Time, time.Now(), time.Second*5)
}
