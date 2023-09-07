package cmd

import (
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

// Env vars
var (
	HealthBlockTimeout = "BLOCK_TIMEOUT"
)

var (
	ConfigFileFlag = &cli.StringFlag{
		Name:  "config",
		Usage: "JSON configuration file",
	}

	VerbosityFlag = &cli.IntFlag{
		Name:  "verbosity",
		Usage: "Supports levels trace (-1) to disable (7)",
		Value: int(zerolog.InfoLevel),
	}

	KeystorePathFlag = &cli.StringFlag{
		Name:  "keystore",
		Usage: "Path to keystore directory",
		Value: "./keys",
	}

	BlockstorePathFlag = &cli.StringFlag{
		Name:  "blockstore",
		Usage: "Specify path for blockstore",
		Value: "./blockstore",
	}

	IsLoaded = &cli.BoolFlag{
		Name:  "isloaded",
		Usage: "Disables loading from blockstore at start. Opts will still be used if specified.",
		Value: true,
	}

	PasswordPathFlag = &cli.StringFlag{
		Name:  "password",
		Usage: "Path to the password file. Passwords in the file must be sorted in order of config chain index.",
		Value: "./password",
	}
)
