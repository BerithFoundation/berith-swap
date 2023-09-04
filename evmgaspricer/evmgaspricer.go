package evmgaspricer

import (
	"context"
	"math/big"
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

func (gasPricer *LondonGasPriceDeterminant) estimateGasLondon(baseFee *big.Int) (*big.Int, *big.Int, error) {
	var maxPriorityFeePerGas *big.Int
	var maxFeePerGas *big.Int

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

	if gasPricer.opts != nil && gasPricer.opts.UpperLimitFeePerGas != nil && maxFeePerGas.Cmp(gasPricer.opts.UpperLimitFeePerGas) == 1 {
		maxPriorityFeePerGas.Sub(gasPricer.opts.UpperLimitFeePerGas, baseFee)
		maxFeePerGas = gasPricer.opts.UpperLimitFeePerGas
	}
	return maxPriorityFeePerGas, maxFeePerGas, nil
}
