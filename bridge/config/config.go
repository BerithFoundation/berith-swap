package config

import (
	cmd "berith-swap/bridge/cmd"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

type Config struct {
	ChainConfig    []RawChainConfig `json:"chains"`
	KeystorePath   string           `json:"keystorePath,omitempty"`
	BlockStorePath string           `json:"blockStorePath"`
	LatestBlock    bool             `json:"latestBlock"`
	Verbosity      zerolog.Level
}

type RawChainConfig struct {
	Idx                int8   `json:"idx"`
	Name               string `json:"name"`
	Type               string `json:"type"`
	Endpoint           string `json:"endpoint"`
	Exchanger          string `json:"exchanger"`
	Erc20Address       string `json:"erc20Address"`
	GasLimit           string `json:"gasLimit"`
	MaxGasPrice        string `json:"maxGasPrice"`
	BlockConfirmations string `json:"blockConfirmations"`
	Password           string
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
		if fresh := ctx.Bool(cmd.LatestBlockFlag.Name); fresh {
			C.LatestBlock = fresh
		}
		if verbosity := ctx.Int64(cmd.VerbosityFlag.Name); zerolog.TraceLevel <= zerolog.Level(verbosity) && zerolog.Level(verbosity) <= zerolog.Disabled {
			C.Verbosity = zerolog.Level(verbosity)
		}
		if pwPath := ctx.String(cmd.PasswordPathFlag.Name); pwPath != "" {
			abPath, err := filepath.Abs(pwPath)
			if err != nil {
				log.Error().Err(err).Msg("cannot get absolute path of password file")
			}
			text, err := os.ReadFile(abPath)
			if err != nil {
				log.Error().Err(err).Msg("cannot read password file")
			}
			lines := strings.Split(string(text), "\n")
			for i, l := range lines {
				C.ChainConfig[i].Password = l
			}
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
