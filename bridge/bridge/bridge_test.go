package bridge

import (
	"berith-swap/bridge/contract"
	"berith-swap/bridge/message"
	"berith-swap/bridge/transaction"
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const (
	testPrivKey = "99a524626eedfb39288b4be1d5533882e3c36cff3a1b4de6f6ab9b45cf1d13b0"
)

var (
	wg            sync.WaitGroup
	gasPrice      = big.NewInt(25000000000)
	defaultTxOpts = transaction.TransactOptions{
		GasLimit: big.NewInt(9000000).Uint64(),
	}
)

func TestBridgeStartStop(t *testing.T) {
	cfg := initTestconfig(t)
	bridge := newTestBridge(t, cfg)

	errCh := make(chan error)
	go func(ch chan error) {
		err := bridge.Start()
		errCh <- err
	}(errCh)

	time.Sleep(5 * time.Second)

	bridge.Stop()

	t.Log(<-errCh)
}

// Receiver를 지정하지 않은 Deposit event를 Sender와 동일한 Receiver로 재 설정하여 처리하는가?
// 테스트 시간 1분 소요
func TestInvalidReceiver(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	cfg := initTestconfig(t)
	senderCfg := cfg.ChainConfig[SenderIdx]
	bridge := newTestBridge(t, cfg)

	bridgeCt, owner := testNewBridgeContract(t, senderCfg)

	var testCases = []struct {
		name    string
		address common.Address
		check   func(err error)
	}{
		{
			name:    "pass",
			address: owner,
			check:   func(err error) { require.NoError(t, err) },
		},
		{
			name:    "pass zero address",
			address: common.Address{},
			check:   func(err error) { require.NoError(t, err) },
		},
		{
			name:    "fail invalid address",
			address: common.HexToAddress("G000000000000000000000000000000000000000"),
			check:   func(err error) { require.Error(t, err) },
		},
	}

	before, err := bridge.rc.erc20Contract.GetBalance(owner)
	require.NoError(t, err)

	sendAmt := big.NewInt(0).Mul(big.NewInt(1), big.NewInt(1e18))

	defaultTxOpts.Value = sendAmt

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := bridgeCt.Deposit(tc.address, defaultTxOpts)
			require.NoError(t, err)

			go bridge.Start()

			checkBalance(t, bridge.rc.erc20Contract, before, sendAmt, owner)

			bridge.Stop()
		})
	}
}

func checkBalance(t *testing.T, erc20 *contract.ERC20Contract, before, amt *big.Int, addr common.Address) bool {
	retry := 50
	for retry > 0 {
		after, err := erc20.GetBalance(addr)
		require.NoError(t, err)
		amtEther := new(big.Int).Div(amt, big.NewInt(1e18))

		check := new(big.Int).Add(before, amtEther).Cmp(after) == 0
		if check {
			erc20.Logger.Debug().Msg("balance checking passed")
			return true
		}
		time.Sleep(5 * time.Second)
		retry--
	}
	t.Fatal("balance checking failed")
	return false
}

// 부하 테스트, 테스트 시간 최대 2분가량 필요
func TestDeposit(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	const cnt = 100
	var wg sync.WaitGroup
	wg.Add(cnt)

	cfg := initTestconfig(t)
	senderCfg := cfg.ChainConfig[SenderIdx]
	bridge := newTestBridge(t, cfg)

	bridgeCt, _ := testNewBridgeContract(t, senderCfg)

	sendAmt := big.NewInt(0).Mul(big.NewInt(1), big.NewInt(1e18))

	defaultTxOpts.Value = sendAmt

	msgCh := make(chan message.DepositMessage, cnt)
	for i := 0; i < cnt; i++ {
		go func(ch chan<- message.DepositMessage) {
			randomBytes := make([]byte, 20)
			_, err := rand.Read(randomBytes)
			if err != nil {
				log.Panicf("generating random account failed. err:%s", err.Error())
			}

			receiver := common.HexToAddress(hex.EncodeToString(randomBytes))
			hash, err := bridgeCt.Deposit(receiver, defaultTxOpts)
			if err != nil {
				log.Panicf("deposit failed. err:%s", err.Error())
			}

			rec, err := bridge.sc.c.EvmClient.Client.TransactionReceipt(context.Background(), *hash)
			require.NoError(t, err)
			ch <- message.DepositMessage{BlockNumber: rec.BlockNumber.Uint64(), Receiver: receiver, Amount: new(big.Int).Div(sendAmt, big.NewInt(1e18))}
			wg.Done()
		}(msgCh)
	}

	wg.Wait()
	close(msgCh)

	wg.Add(cnt)
	for m := range msgCh {
		go func(msg message.DepositMessage) {
			err := bridge.rc.SendToken(msg)
			require.NoError(t, err)

			bal, err := bridge.rc.erc20Contract.GetBalance(msg.Receiver)
			require.NoError(t, err)
			require.Equal(t, sendAmt.Cmp(bal), 1)
			wg.Done()
		}(m)
	}
	wg.Wait()
}
