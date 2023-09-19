package contract

import (
	"berith-swap/bridge/transaction"
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBridgeLowGasPrice(t *testing.T) {
	bridgeCtr := newTestBridgeContract(t, BerithBaseFee)

	gp, err := bridgeCtr.client.SuggestGasPrice(context.Background())
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
			gasPrice: new(big.Int).Sub(gp, common.Big1),
			expect: func(t *testing.T, err error, hash *common.Hash) {
				require.Error(t, err)
				require.Nil(t, hash)
			},
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := bridgeCtr.Deposit(common.Address{}, transaction.TransactOptions{
				GasLimit: transaction.DefaultGasLimit,
				GasPrice: tc.gasPrice,
				Value:    new(big.Int).Mul(common.Big1, big.NewInt(1e18)),
			})
			tc.expect(t, err, hash)
		})
	}
}

func TestZeroGasLimitBridge(t *testing.T) {

	bridgeCtr := newTestBridgeContract(t, BerithBaseFee)
	hash, err := bridgeCtr.Deposit(common.Address{}, transaction.TransactOptions{
		GasLimit: 0,
		GasPrice: big.NewInt(0),
		Value:    big.NewInt(0),
	}) // Berith 네트워크는 basefee 개념이 없어 LondonGasPriceClient 수 없다.

	require.NotNil(t, hash)
	require.NoError(t, err)
}
