# Berith swap

### Dependency
- Go 1.20
- [SQLC](https://docs.sqlc.dev/en/latest/overview/install.html)
- HardHat `npm install --save-dev hardhat`
  
### 설정
```
config.json
{
  "chains": [
    {
      "idx": 0,
      "name": "berith",
      "endpoint": "https://bers.berith.co/",
      "owner": "Swap Owner Address",
      "swapAddress": "Swap Contract Address",
      "gasLimit": "3000000",
      "maxGasPrice": "1000000000",
      "blockConfirmations": "10" // 최근 블록 - 탐색하려는 블록의 필요 간격
    },
    {
      "idx": 1,
      "name": "klaytn",
      "endpoint": "https://public-en-cypress.klaytn.net",
      "owner": "ERC20 Owner Address",
      "erc20Address": "ERC20 Contract Address",
      "gasLimit": "9000000",
      "maxGasPrice": "10000000000",
      "blockConfirmations": "10"
    }
  ],
  "keystorePath": "",
  "blockStorePath": "",
  "dbSource": "원격 DB Table의 접속정보. ex) user:password@tcp(url)/table"
}
```
### 컨트랙트 배포
`Make deploy`

Klaytn Mainnet에 ERC20 토큰 컨트랙트와 Berith Mainnet에 Swap 컨트랙트를 각각 배포

### ORM 생성
`Make sqlc`

`store/query` 에서 작성한 SQL Query를 Go 코드로 생성


### 실행
```
실행 flags

NAME:
   berith-swap - BerithSwap

USAGE:
   berith-swap [global options] command [command options] [arguments...]

VERSION:
   0.0.1

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config value      config.json 파일의 경로를 지정합니다.
   --verbosity value   디버깅 레벨을 조절합니다 Trace (-1) -> Disable (7) (default: 1)
   --keystore value    키파일들이 위치한 경로를 지정합니다. (default: "./keys")
   --password value    키파일에 해당하는 비밀번호가 저장된 파일의 경로를 지정합니다. Sender는 첫줄, Receiver는 두번째 줄에 기입합니다. (default: "./password")
   --blockstore value  blockstore 경로를 지정합니다. (default: "./blockstore")
   --load              만약 true라면, blockstore에서 마지막으로 Deposit된 블록 번호를 로드하여 해당 블록부터 Fetching을 실행합니다. (default: true)
   --help, -h          show help
   --version, -v       print the version

COPYRIGHT:
   Copyright 2023 Berith foundation Authors
```

### 디버그

```
.vscode/launch.json

{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Package",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}",
            "args": ["--config=config.json_경로","--keystore=Keyfile_디렉토리_경로","--password=password_파일_경로","--blockstore=blockstore_경로_지정","--verbosity=-1","--load=false"]
        }
    ]
}
```