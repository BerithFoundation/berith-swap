package evmgaspricer

import (
	"context"
	"math/big"

	"github.com/rs/zerolog/log"
)

type LondonGasPriceDeterminant struct {
	client LondonGasClient
	opts   *GasPricerOpts
}

func NewLondonGasPriceClient(client LondonGasClient, opts *GasPricerOpts) *LondonGasPriceDeterminant {
	return &LondonGasPriceDeterminant{client: client, opts: opts}
}

func (gasPricer *LondonGasPriceDeterminant) GasPrice(priority *uint8) ([]*big.Int, error) {
	baseFee, err := gasPricer.client.BaseFee()
	if err != nil {
		return nil, err
	}
	gasPrices := make([]*big.Int, 2)
	if baseFee == nil {
		staticGasPricer := NewStaticGasPriceDeterminant(gasPricer.client, gasPricer.opts)
		return staticGasPricer.GasPrice(nil)
	}
	gasTipCap, gasFeeCap, err := gasPricer.estimateGasLondon(baseFee)
	log.Info().Msgf("Suggested Max Fee: %s, Gas tip: %s", gasFeeCap.String(), gasTipCap.String())
	if err != nil {
		return nil, err
	}
	gasPrices[0] = gasTipCap
	gasPrices[1] = gasFeeCap
	return gasPrices, nil
}

func (gasPricer *LondonGasPriceDeterminant) SetClient(client LondonGasClient) {
	gasPricer.client = client
}
func (gasPricer *LondonGasPriceDeterminant) SetOpts(opts *GasPricerOpts) {
	gasPricer.opts = opts
}

const TwoAndTheHalfGwei = 2500000000

// estimateGasLondon은 maxPriorityFeePerGas와 maxFeePerGas를 계산한다.
//
// baseFee - 기본 수수료 (거래를 제출하기 위해 지불해야할 최소한의 수수료, 이전 블록에서 사용된 가스량에 따라 결정됨)
//
// maxPriorityFeePerGas - EIP-1559 하드포크로 인하여 baseFee가 소각되기 때문에 채굴자들이 제출한 거래를 포함해주도록 추가로 지불해야 하는 팁
//
// maxFeePerGas - baseFee + maxPriorityFeePerGas
func (gasPricer *LondonGasPriceDeterminant) estimateGasLondon(baseFee *big.Int) (*big.Int, *big.Int, error) {
	var maxPriorityFeePerGas *big.Int // 최대 지불 가능한 팁
	var maxFeePerGas *big.Int         // basefee + 최대 팁

	if gasPricer.opts != nil && gasPricer.opts.UpperLimitFeePerGas != nil && gasPricer.opts.UpperLimitFeePerGas.Cmp(baseFee) < 0 {
		maxPriorityFeePerGas = big.NewInt(TwoAndTheHalfGwei)
		maxFeePerGas = new(big.Int).Add(baseFee, maxPriorityFeePerGas)
		return maxPriorityFeePerGas, maxFeePerGas, nil
	}

	maxPriorityFeePerGas, err := gasPricer.client.SuggestGasTipCap(context.TODO())
	if err != nil {
		return nil, nil, err
	}
	maxFeePerGas = new(big.Int).Add(
		maxPriorityFeePerGas,
		new(big.Int).Mul(baseFee, big.NewInt(2)),
	)

	// gaspricer에 설정된 최대 가스 지불 제한량보다 maxFeePerGas이 더 크면
	// 설정된 제한량에서 base fee를 뺀값을 maxPriorityFeePerGas로 재설정
	if gasPricer.opts != nil && gasPricer.opts.UpperLimitFeePerGas != nil && maxFeePerGas.Cmp(gasPricer.opts.UpperLimitFeePerGas) == 1 {
		maxPriorityFeePerGas.Sub(gasPricer.opts.UpperLimitFeePerGas, baseFee) // basefee가 30이고 UpperLimitFeePerGas가 35면 maxPriorityFeePerGas는 5로 재설정
		maxFeePerGas = gasPricer.opts.UpperLimitFeePerGas                     // basefee는 30이고, maxPriorityFeePerGas를 5로 설정해 주었기 때문에 maxFeePerGas는 초기 설정대로 35
	}
	return maxPriorityFeePerGas, maxFeePerGas, nil
}
