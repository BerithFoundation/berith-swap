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
	IsLoaded       bool
	Verbosity      zerolog.Level
}

type RawChainConfig struct {
	Idx                int8   `json:"idx"`
	Name               string `json:"name"`
	Endpoint           string `json:"endpoint"`
	Owner              string `json:"owner"`
	BridgeAddress      string `json:"bridgeAddress"`
	Erc20Address       string `json:"erc20Address"`
	GasLimit           string `json:"gasLimit"`
	MaxGasPrice        string `json:"maxGasPrice"`
	BlockConfirmations string `json:"blockConfirmations"`
	Password           string
}

const (
	DefaultConfigPath = "./config.json"
)

func GetConfig(ctx *cli.Context) (*Config, error) {
	cfg := new(Config)

	path := DefaultConfigPath
	if file := ctx.String(cmd.ConfigFileFlag.Name); file != "" {
		path = file
	}
	err := loadConfig(path, cfg)
	if err != nil {
		log.Warn().Err(err).Msg("cannot parse json file")
		return nil, err
	}
	if ksPath := ctx.String(cmd.KeystorePathFlag.Name); ksPath != "" {
		cfg.KeystorePath = ksPath
	}
	if store := ctx.String(cmd.BlockstorePathFlag.Name); store != "" {
		cfg.BlockStorePath = store
	}
	if isLoaded := ctx.Bool(cmd.LoadFlag.Name); isLoaded {
		cfg.IsLoaded = isLoaded
	}
	if verbosity := ctx.Int64(cmd.VerbosityFlag.Name); zerolog.TraceLevel <= zerolog.Level(verbosity) && zerolog.Level(verbosity) <= zerolog.Disabled {
		cfg.Verbosity = zerolog.Level(verbosity)
	}
	if pwPath := ctx.String(cmd.PasswordPathFlag.Name); pwPath != "" {
		lines, err := ParsePasswordFile(pwPath)
		if err != nil {
			log.Error().Err(err).Msg("cannot parse passsword file")
		}
		for i, l := range lines {
			cfg.ChainConfig[i].Password = l
		}
	}

	return cfg, nil
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

func ParsePasswordFile(pwPath string) ([]string, error) {
	abPath, err := filepath.Abs(pwPath)
	if err != nil {
		log.Error().Err(err).Msg("cannot get absolute path of password file")
		return nil, err
	}
	text, err := os.ReadFile(abPath)
	if err != nil {
		log.Error().Err(err).Msgf("cannot read password file. path:%s", abPath)
		return nil, err
	}
	lines := strings.Split(string(text), "\n")

	return lines, nil
}
