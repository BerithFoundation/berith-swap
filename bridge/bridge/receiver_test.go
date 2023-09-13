package bridge

import (
	"berith-swap/bridge/blockstore"
	"berith-swap/bridge/config"
	"berith-swap/bridge/message"
	"berith-swap/bridge/store/mariadb"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStoreHistory(t *testing.T) {
	ch := make(chan message.DepositMessage)
	f, err := os.Open("../../run_test/config.json")
	require.NoError(t, err)

	cfg := new(config.Config)
	require.NoError(t, json.NewDecoder(f).Decode(cfg))

	cfg.KeystorePath = "../../run_test/keys"

	lines, err := config.ParsePasswordFile(configDir + "password")
	require.NoError(t, err)

	chainCfg := cfg.ChainConfig[ReceiverIdx]
	cfg.ChainConfig[ReceiverIdx].Password = lines[ReceiverIdx]

	bs, err := blockstore.NewBlockstore("../../run_test/blockstore", chainCfg.Name)
	require.NoError(t, err)
	require.NotNil(t, bs)

	rc := NewReceiverChain(ch, cfg, ReceiverIdx, bs)

	err = rc.store.CreateSwapHistoryTx(context.Background(), mariadb.CreateBersSwapHistoryParams{
		SenderTxHash:   "senderTx",
		ReceiverTxHash: "receiverTx",
		Amount:         1,
	})

	require.NoError(t, err)

	hist, err := rc.store.GetBersSwapHistory(context.Background(), "senderTx")
	require.NoError(t, err)
	require.Equal(t, hist.Amount, int64(1))
	require.Equal(t, hist.ReceiverTxHash, "receiverTx")
	require.Equal(t, hist.SenderTxHash, "senderTx")
	require.WithinDuration(t, hist.CreatedAt.Time, time.Now(), time.Second*5)
}
