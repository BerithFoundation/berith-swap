package evmgaspricer

import (
	"context"
	"math/big"
)

type LondonGasClient interface {
	GasPriceClient
	BaseFee() (*big.Int, error)
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
}

type GasPriceClient interface {
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
}

type GasPricerOpts struct {
	UpperLimitFeePerGas *big.Int
	GasPriceFactor      *big.Float
	Args                []interface{}
}

func multiplyGasPrice(gasEstimate *big.Int, gasMultiplier *big.Float) *big.Int {
	gasEstimateFloat := new(big.Float).SetInt(gasEstimate)
	result := gasEstimateFloat.Mul(gasEstimateFloat, gasMultiplier)
	gasPrice := new(big.Int)
	result.Int(gasPrice)
	return gasPrice
}
