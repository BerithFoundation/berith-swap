package contract

import (
	"berith-swap/bridge/connection"
	"berith-swap/bridge/keypair"
	"berith-swap/bridge/transaction"
	"berith-swap/logger"
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestZeroGasLimitOptsERC20(t *testing.T) {

	testCases := []struct {
		name        string
		gasPayLimit *big.Int
		expect      func(*testing.T, error, transaction.ContractCallerDispatcher, common.Hash)
	}{
		{
			name:        "default gas pay limit",
			gasPayLimit: KlaytnBaseFee,
			expect: func(t *testing.T, err error, client transaction.ContractCallerDispatcher, hash common.Hash) {
				require.NoError(t, err)

				receipt, err := client.TransactionReceipt(context.Background(), hash)
				require.NoError(t, err)
				require.Equal(t, receipt.EffectiveGasPrice.Cmp(KlaytnBaseFee), 0)
			},
		},
		{
			name:        "half gas pay limit",
			gasPayLimit: new(big.Int).Div(KlaytnBaseFee, big.NewInt(2)),
			expect: func(t *testing.T, err error, client transaction.ContractCallerDispatcher, hash common.Hash) {
				require.NoError(t, err)
				receipt, err := client.TransactionReceipt(context.Background(), hash)
				require.NoError(t, err)
				require.Equal(t, receipt.EffectiveGasPrice.Cmp(KlaytnBaseFee), 0)
			},
		},
		{
			name:        "triple gas pay limit",
			gasPayLimit: new(big.Int).Mul(KlaytnBaseFee, big.NewInt(3)),
			expect: func(t *testing.T, err error, client transaction.ContractCallerDispatcher, hash common.Hash) {
				require.NoError(t, err)
				receipt, err := client.TransactionReceipt(context.Background(), hash)
				require.NoError(t, err)
				require.GreaterOrEqual(t, receipt.EffectiveGasPrice.Cmp(KlaytnBaseFee), 0)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			erc20Ctr := newTestERC20Contract(t, tc.gasPayLimit) // Klaytn network는 basefee가 있어 StaticGasPriceClient를 사용해야 한다.
			hash, err := erc20Ctr.Transfer(erc20Ctr.Contract.client.From(), big.NewInt(1), transaction.TransactOptions{
				GasLimit: transaction.DefaultGasLimit,
				GasPrice: big.NewInt(0),
				Value:    big.NewInt(0),
			})

			tc.expect(t, err, erc20Ctr.client, *hash)
		})
	}

}

func TestErc20LowGasPrice(t *testing.T) {
	erc20Ctr := newTestERC20Contract(t, KlaytnBaseFee)

	block, err := erc20Ctr.client.LatestBlock()
	require.NoError(t, err)

	gp := block.BaseFee()

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
			gasPrice: new(big.Int).Sub(gp, common.Big1),
			expect: func(t *testing.T, err error, hash *common.Hash) {
				require.Error(t, err)
				require.Nil(t, hash)
			},
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := erc20Ctr.Transfer(erc20Ctr.client.From(), common.Big1, transaction.TransactOptions{
				GasLimit: transaction.DefaultGasLimit,
				GasPrice: tc.gasPrice,
			})
			tc.expect(t, err, hash)
		})
	}
}

func TestOnlyOwnerCanPause(t *testing.T) {
	cfg := initTestconfig(t)
	erc20Ctr := newTestERC20Contract(t, KlaytnBaseFee)

	newKp := testGenerateKeypair(t)

	newLogger := logger.NewLogger(zerolog.DebugLevel, "Owner Pause")
	newClient, err := connection.NewEvmClient(newKp, cfg.ChainConfig[ReceiverIdx].Endpoint, &newLogger)
	require.NoError(t, err)

	tran, err := InitializeTransactor(KlaytnBaseFee, transaction.NewTransaction, newClient)
	require.NoError(t, err)

	notOwnerCtConn := NewERC20Contract(newClient, erc20Ctr.Contract.contractAddress, tran, &newLogger)

	notOwnerAddr := newKp.CommonAddress()
	sendAmt := big.NewInt(1e17) // ownable contract의 연산도 거쳐 tx fee가 추가로 발생함
	hash, err := erc20Ctr.Transact(&notOwnerAddr, nil, transaction.TransactOptions{
		GasLimit: transaction.DefaultGasLimit,
		GasPrice: KlaytnBaseFee,
		Value:    sendAmt,
	})
	newLogger.Debug().Msgf("transfer klay hex: %s", hash.Hex())
	require.NoError(t, err)
	require.NotNil(t, hash)

	bal, err := newClient.Client.BalanceAt(context.Background(), notOwnerAddr, nil)
	require.NoError(t, err)
	require.Equal(t, sendAmt.Cmp(bal), 0)

	hash, err = notOwnerCtConn.Pause(transaction.DefaultTransactionOptions) // should fail
	require.Error(t, err)
	require.Nil(t, hash)
}

// Pause 상태일 때 transfer가 되지 않는지 확인
func TestErc20Pausable(t *testing.T) {
	erc20Ctr := newTestERC20Contract(t, KlaytnBaseFee)

	checkReceipt := func(hash *common.Hash) uint64 {
		receipt, _ := erc20Ctr.client.TransactionReceipt(context.Background(), *hash)
		return receipt.Status
	}

	newKp, err := keypair.GenerateKeyPair(testAccount, testKeyDir, testPW)
	require.NoError(t, err)

	newAddr := newKp.CommonAddress()

	state, err := erc20Ctr.GetPauseState()
	require.NoError(t, err)
	require.NotNil(t, state)

	var hash *common.Hash

	if !*state {
		hash, _ = erc20Ctr.Transfer(newAddr, common.Big1, transaction.DefaultTransactionOptions)
		require.Equal(t, checkReceipt(hash), uint64(1), hash.Hex())

		hash, _ = erc20Ctr.Pause(transaction.DefaultTransactionOptions)
		require.Equal(t, checkReceipt(hash), uint64(1), hash.Hex())
	}

	hash, err = erc20Ctr.Transfer(newAddr, common.Big1, transaction.DefaultTransactionOptions)
	require.Nil(t, hash)
	require.Error(t, err) // Should Fail

	hash, _ = erc20Ctr.UnPause(transaction.DefaultTransactionOptions)
	require.Equal(t, checkReceipt(hash), uint64(1), hash.Hex())

	hash, _ = erc20Ctr.Transfer(newAddr, common.Big1, transaction.DefaultTransactionOptions)
	require.Equal(t, checkReceipt(hash), uint64(1), hash.Hex())
}
