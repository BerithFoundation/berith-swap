package main

import (
	"berith-swap/bridge/bridge"
	"berith-swap/bridge/cmd"
	"berith-swap/bridge/config"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

const (
	Version = "0.0.1"
)

var app = cli.NewApp()

var cliFlags = []cli.Flag{
	cmd.ConfigFileFlag,
	cmd.VerbosityFlag,
	cmd.KeystorePathFlag,
	cmd.PasswordPathFlag,
	cmd.BlockstorePathFlag,
	cmd.LoadFlag,
	cmd.DBSourceFlag,
}

func init() {
	app.Action = run
	app.Name = "berith-swap"
	app.Usage = "BerithSwap"
	app.Copyright = "Copyright 2023 Berith foundation Authors"
	app.Version = Version
	app.Flags = append(app.Flags, cliFlags...)

}

func main() {
	if err := app.Run(os.Args); err != nil {
		log.Error().Err(err).Msg("berith swap got error during running.. shutdown")
		os.Exit(1)
	}
}

func run(ctx *cli.Context) error {
	cfg, err := config.GetConfig(ctx)
	if err != nil {
		return err
	}
	b := bridge.NewBridge(cfg)
	return b.Start()
	//TODO: receiver tx 실패 시 sender 블록 스토어 롤백 적용하기
}
