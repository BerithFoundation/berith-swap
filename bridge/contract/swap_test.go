package contract

import (
	"berith-swap/bridge/keypair"
	"berith-swap/bridge/transaction"
	"context"
	"encoding/hex"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestSwapLowGasPrice(t *testing.T) {
	cfg := initTestconfig(t)
	kp, err := keypair.GenerateKeyPair(cfg.ChainConfig[SenderIdx].Owner, cfg.KeystorePath, cfg.ChainConfig[SenderIdx].Password)
	require.NoError(t, err)

	swapCtr := newTestSwapContract(t, kp)

	gp, err := swapCtr.client.SuggestGasPrice(context.Background())
	require.NoError(t, err)

	testCase := []struct {
		name     string
		gasPrice *big.Int
		expect   func(*testing.T, error, *common.Hash)
	}{
		{
			name:     "suggested gas price",
			gasPrice: gp,
			expect: func(t *testing.T, err error, hash *common.Hash) {
				require.NoError(t, err)
				require.NotNil(t, hash)
			},
		},
		{
			name:     "under gas price",
			gasPrice: common.Big1, //Berith chain은 basefee가 없고 tx 경쟁이 낮아 gas price가 낮아도 통과될 것으로 예상됨
			expect: func(t *testing.T, err error, hash *common.Hash) {
				require.NoError(t, err)
				require.Nil(t, hash)
			},
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := swapCtr.Deposit(common.Address{}, transaction.TransactOptions{
				GasLimit: transaction.DefaultGasLimit,
				GasPrice: tc.gasPrice,
				Value:    new(big.Int).Mul(common.Big1, big.NewInt(1e18)),
			})
			tc.expect(t, err, hash)
		})
	}
}

// Test 소요시간 1분
func TestSwapWithdraw(t *testing.T) {
	cfg := initTestconfig(t)
	kp, err := keypair.GenerateKeyPair(cfg.ChainConfig[SenderIdx].Owner, cfg.KeystorePath, cfg.ChainConfig[SenderIdx].Password)
	require.NoError(t, err)

	swapCtr := newTestSwapContract(t, kp)

	testKp := testGenerateKeypair(t)
	defer removeTestKeyStore(t)

	testSwapCtr := newTestSwapContract(t, testKp)
	testFrom := testSwapCtr.client.From()

	sendBersHash, err := swapCtr.Transact(&testFrom, nil, transaction.TransactOptions{
		GasLimit: transaction.DefaultGasLimit,
		GasPrice: BerithGasPrice,
		Value:    new(big.Int).Mul(common.Big1, big.NewInt(1.1e18)),
	})
	require.NoError(t, err)
	require.NotNil(t, sendBersHash)

	type testCase struct {
		name   string
		swap   *SwapContract
		expect func(*testing.T, error)
	}
	testCases := []testCase{
		{
			name: "owner",
			swap: swapCtr,
			expect: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "not owner",
			swap: testSwapCtr,
			expect: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
	}

	var wg sync.WaitGroup
	wg.Add(len(testCases))

	for _, tc := range testCases {
		go func(c testCase, wait *sync.WaitGroup) {
			t.Run(c.name, func(t *testing.T) {
				from := c.swap.client.From()
				_, err := c.swap.Deposit(from, transaction.TransactOptions{
					GasLimit: transaction.DefaultGasLimit,
					GasPrice: common.Big0,
					Value:    new(big.Int).Mul(common.Big1, big.NewInt(1e18)),
				})
				require.NoError(t, err)

				_, err = c.swap.TransferFunds(transaction.DefaultTransactionOptions)
				c.expect(t, err)
				wg.Done()
			})
		}(tc, &wg)
	}

	wg.Wait()
}

func TestGetPrivateKey(t *testing.T) {
	cfg := initTestconfig(t)
	kp, err := keypair.GenerateKeyPair(cfg.ChainConfig[SenderIdx].Owner, cfg.KeystorePath, cfg.ChainConfig[SenderIdx].Password)
	require.NoError(t, err)

	t.Log(kp.CommonAddress())
	t.Log(hex.EncodeToString(kp.Encode()))
}
