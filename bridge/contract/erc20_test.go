package contract

import (
	"berith-swap/bridge/transaction"
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestZeroGasLimitERC20(t *testing.T) {

	testCases := []struct {
		name     string
		gasPrice *big.Int
		expect   func(*testing.T, error, transaction.ContractCallerDispatcher, common.Hash)
	}{
		{
			name:     "default gas price",
			gasPrice: KlaytnBaseFee,
			expect: func(t *testing.T, err error, client transaction.ContractCallerDispatcher, hash common.Hash) {
				require.NoError(t, err)

				receipt, err := client.TransactionReceipt(context.Background(), hash)
				require.NoError(t, err)
				require.Equal(t, receipt.EffectiveGasPrice.Cmp(KlaytnBaseFee), 0)
			},
		},
		{
			name:     "half gas price",
			gasPrice: new(big.Int).Div(KlaytnBaseFee, big.NewInt(2)),
			expect: func(t *testing.T, err error, client transaction.ContractCallerDispatcher, hash common.Hash) {
				require.NoError(t, err)
				receipt, err := client.TransactionReceipt(context.Background(), hash)
				require.NoError(t, err)
				require.Equal(t, receipt.EffectiveGasPrice.Cmp(KlaytnBaseFee), 0)
			},
		},
		{
			name:     "triple gas price",
			gasPrice: new(big.Int).Mul(KlaytnBaseFee, big.NewInt(3)), // gasFeeCap: 75Gwei, gasTipCap: 50Gwei
			expect: func(t *testing.T, err error, client transaction.ContractCallerDispatcher, hash common.Hash) {
				require.NoError(t, err)
				receipt, err := client.TransactionReceipt(context.Background(), hash)
				require.NoError(t, err)
				require.Equal(t, receipt.EffectiveGasPrice.Cmp(KlaytnBaseFee), 0) // Klaytn은 tip지불을 위해 basefee보다 높은 gasPrice를 제시하여도 basefee로 Tx가 실행된다.
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			erc20Ctr := newTestERC20Contract(t, tc.gasPrice) // Klaytn network는 basefee가 있어 StaticGasPriceClient를 사용해야 한다.
			hash, err := erc20Ctr.Transfer(erc20Ctr.Contract.client.From(), big.NewInt(1), transaction.TransactOptions{
				GasLimit: 0, // GasLimit이 0일경우엔 Transct하는 과정에서 Defalult TransacOptions에 merge됨
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
