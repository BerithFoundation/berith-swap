package bridge

import (
	"berith-swap/bridge/blockstore"
	"berith-swap/bridge/keypair"
	"berith-swap/bridge/message"
	"berith-swap/bridge/store/mariadb"
	"context"
	"crypto/rand"
	"database/sql"
	"math/big"
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
	receiver, err := keypair.GenerateKeyPair(testAccount, testKeyDir, testPW)
	require.NoError(t, err)

	ch <- message.DepositMessage{
		BlockNumber:  big.NewInt(1).Uint64(),
		Amount:       big.NewInt(1),
		Sender:       receiver.CommonAddress(),
		Receiver:     receiver.CommonAddress(),
		SenderTxHash: senderTx,
	}

	var hist mariadb.BersSwapHist
	for i := 0; i < 10; i++ {
		hist, err = rc.store.GetBersSwapHistory(context.Background(), senderTx)
		if err == sql.ErrNoRows {
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}
	require.NoError(t, err)
	require.Equal(t, hist.Amount, int64(1))
	require.NotEmpty(t, hist.ReceiverTxHash)
	require.Equal(t, hist.SenderTxHash, senderTx)
	require.WithinDuration(t, hist.CreatedAt.Time, time.Now().UTC(), time.Second*5)
}
