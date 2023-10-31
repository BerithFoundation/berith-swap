package cmd

import (
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

var (
	ConfigFileFlag = &cli.StringFlag{
		Name:  "config",
		Usage: "config.json 파일의 경로를 지정합니다.",
	}

	VerbosityFlag = &cli.IntFlag{
		Name:  "verbosity",
		Usage: "디버깅 레벨을 조절합니다 Trace (-1) -> Disable (7)",
		Value: int(zerolog.InfoLevel),
	}

	KeystorePathFlag = &cli.StringFlag{
		Name:  "keystore",
		Usage: "키파일들이 위치한 경로를 지정합니다.",
		Value: "./keys",
	}

	BlockstorePathFlag = &cli.StringFlag{
		Name:  "blockstore",
		Usage: "blockstore 경로를 지정합니다.",
		Value: "./blockstore",
	}

	LoadFlag = &cli.BoolFlag{
		Name:  "load",
		Usage: "만약 true라면, blockstore에서 마지막으로 Deposit된 블록 번호를 로드하여 해당 블록부터 Fetching을 실행합니다.",
		Value: true,
	}

	PasswordPathFlag = &cli.StringFlag{
		Name:     "password",
		Required: true,
		Usage:    "키파일에 해당하는 비밀번호가 저장된 파일의 경로를 지정합니다. Sender는 첫줄, Receiver는 두번째 줄에 기입합니다.",
		Value:    "./password",
	}

	DBSourceFlag = &cli.StringFlag{
		Name:  "dbsource",
		Usage: "원격 DB Table의 접속정보를 지정합니다. ex) user:password@tcp(url)/table",
		Value: "",
	}
)
