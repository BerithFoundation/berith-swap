package chain

import (
	"berith-swap/blockstore"
	"berith-swap/config"
	"berith-swap/util"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

type Chain struct {
	Name                 string                 `json:"name"`
	Endpoint             string                 `json:"endpoint"`
	TokenContractAddress common.Address         `json:"tokenContractAddress"`
	BlockStore           *blockstore.Blockstore `json:"blockStore"`
	GasLimit             *big.Int               `json:"gasLimit"`
	MaxGasPrice          *big.Int               `json:"maxGasPrice"`
	BlockConfirmations   int                    `json:"blockConfirmations"`
}

func NewChain(cfg *config.RawChainConfig, bsPath, ksPath, password string) (*Chain, error) {
	newChain := Chain{
		Name:                 cfg.Name,
		Endpoint:             cfg.Endpoint,
		TokenContractAddress: common.HexToAddress(cfg.TokenContractAddress),
	}
	gl, err := util.StringToBig(cfg.GasLimit, 10)
	if err != nil {
		return nil, err
	}
	newChain.GasLimit = gl

	gp, err := util.StringToBig(cfg.MaxGasPrice, 10)
	if err != nil {
		return nil, err
	}
	newChain.MaxGasPrice = gp

	kp, err := SendToPasswordKeypair(cfg.From, ksPath, password)
	if err != nil {
		return nil, err
	}

	newChain.setupBlockstore(bsPath)

}

func (c *Chain) setupBlockstore(bsPath string) {
	bs, err := blockstore.NewBlockstore(bsPath, c.Name)
	if err != nil {
		log.Error().Err(err).Msg("cannot initialize blockstore")
	}

	c.BlockStore = bs
}
