package config

import (
	"berith-swap/cmd"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

type Config struct {
	Chains         []RawChainConfig `json:"chains"`
	KeystorePath   string           `json:"keystorePath,omitempty"`
	BlockStorePath string           `json:"blockStorePath"`
	FreshStart     bool             `json:"freshStart"`
}

type RawChainConfig struct {
	Name                 string `json:"name"`
	Type                 string `json:"type"`
	Endpoint             string `json:"endpoint"`
	From                 string `json:"from"`
	Password             string `json:"password"`
	TokenContractAddress string `json:"tokenContractAddress"`
	GasLimit             string `json:"gasLimit"`
	MaxGasPrice          string `json:"maxGasPrice"`
	BlockConfirmations   string `json:"blockConfirmations"`
}

const (
	DefaultConfigPath = "./config.json"
)

var (
	C *Config
)

func GetConfig(ctx *cli.Context) (*Config, error) {
	if C != nil {
		path := DefaultConfigPath
		if file := ctx.String(cmd.ConfigFileFlag.Name); file != "" {
			path = file
		}
		err := loadConfig(path, C)
		if err != nil {
			log.Warn().Err(err).Msg("err loading json file")
			return C, err
		}
		if ksPath := ctx.String(cmd.KeystorePathFlag.Name); ksPath != "" {
			C.KeystorePath = ksPath
		}
		if store := ctx.String(cmd.BlockstorePathFlag.Name); store != "" {
			C.BlockStorePath = store
		}
		if fresh := ctx.Bool(cmd.FreshStartFlag.Name); fresh {
			C.FreshStart = fresh
		}
		return C, nil
	}
	return C, nil
}

func loadConfig(file string, config *Config) error {
	ext := filepath.Ext(file)
	fp, err := filepath.Abs(file)
	if err != nil {
		return err
	}

	log.Debug().Any("path", filepath.Clean(fp)).Msg("Loading configuration")

	f, err := os.Open(filepath.Clean(fp))
	if err != nil {
		return err
	}

	if ext == ".json" {
		if err = json.NewDecoder(f).Decode(&config); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unrecognized extention: %s", ext)
	}

	return nil
}
