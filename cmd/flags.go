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
		Value: "",
	}

	FreshStartFlag = &cli.BoolFlag{
		Name:  "fresh",
		Usage: "Disables loading from blockstore at start. Opts will still be used if specified.",
	}

	LatestBlockFlag = &cli.BoolFlag{
		Name:  "latest",
		Usage: "Overrides blockstore and start block, starts from latest block",
	}
	PasswordPathFlag = &cli.StringFlag{
		Name:  "password",
		Usage: "Path to the password file. Passwords in the file must be sorted in order of config chain index.",
		Value: "./keys/password",
	}
)

var (
	Sr25519Flag = &cli.BoolFlag{
		Name:  "sr25519",
		Usage: "Specify account/key type as sr25519.",
	}
	Secp256k1Flag = &cli.BoolFlag{
		Name:  "secp256k1",
		Usage: "Specify account/key type as secp256k1.",
	}
)

var (
	EthereumImportFlag = &cli.BoolFlag{
		Name:  "ethereum",
		Usage: "Import an existing ethereum keystore, such as from geth.",
	}
	PrivateKeyFlag = &cli.StringFlag{
		Name:  "privateKey",
		Usage: "Import a hex representation of a private key into a keystore.",
	}
	SubkeyNetworkFlag = &cli.StringFlag{
		Name:        "network",
		Usage:       "Specify the network to use for the address encoding (substrate/polkadot/centrifuge)",
		DefaultText: "substrate",
	}
)

// Test Setting Flags
var (
	TestKeyFlag = &cli.StringFlag{
		Name:  "testkey",
		Usage: "Applies a predetermined test keystore to the chains.",
	}
)
