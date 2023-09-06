package evmgaspricer

import (
	"context"
	"math/big"

	"github.com/rs/zerolog/log"
)

type StaticGasPriceDeterminant struct {
	client GasPriceClient
	opts   *GasPricerOpts
}

func NewStaticGasPriceDeterminant(client GasPriceClient, opts *GasPricerOpts) *StaticGasPriceDeterminant {
	return &StaticGasPriceDeterminant{client: client, opts: opts}

}

func (gasPricer *StaticGasPriceDeterminant) SetClient(client LondonGasClient) {
	gasPricer.client = client
}
func (gasPricer *StaticGasPriceDeterminant) SetOpts(opts *GasPricerOpts) {
	gasPricer.opts = opts
}

func (gasPricer *StaticGasPriceDeterminant) GasPrice(priority *uint8) ([]*big.Int, error) {
	gp, err := gasPricer.client.SuggestGasPrice(context.TODO())
	log.Debug().Msgf("Suggested GP %s", gp.String())
	if err != nil {
		return nil, err
	}
	if gasPricer.opts != nil {
		if gasPricer.opts.GasPriceFactor != nil {
			gp = multiplyGasPrice(gp, gasPricer.opts.GasPriceFactor)
		}
		if gasPricer.opts.UpperLimitFeePerGas != nil {
			if gp.Cmp(gasPricer.opts.UpperLimitFeePerGas) == 1 {
				gp = gasPricer.opts.UpperLimitFeePerGas
			}
		}
	}
	gasPrices := make([]*big.Int, 1)
	gasPrices[0] = gp
	return gasPrices, nil
}
