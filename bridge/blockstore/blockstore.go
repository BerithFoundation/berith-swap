package blockstore

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
)

const PathPostfix = "berith-swap/blockstore"

type Blockstore struct {
	path      string
	fullPath  string
	chainName string
}

func NewBlockstore(path, chainName string) (*Blockstore, error) {
	fileName := getFileName(chainName)
	if path == "" {
		def, err := getDefaultPath()
		if err != nil {
			return nil, err
		}
		path = def
	}

	return &Blockstore{
		path:      path,
		fullPath:  filepath.Join(path, fileName),
		chainName: chainName,
	}, nil
}

func (b *Blockstore) StoreBlock(block *big.Int) error {
	if _, err := os.Stat(b.path); os.IsNotExist(err) {
		errr := os.MkdirAll(b.path, os.ModePerm)
		if errr != nil {
			return errr
		}
	}

	// Write bytes to file
	data := []byte(block.String())
	err := os.WriteFile(b.fullPath, data, 0600)
	if err != nil {
		return err
	}
	return nil
}

func (b *Blockstore) TryLoadLatestBlock() (*big.Int, error) {
	exists, err := fileExists(b.fullPath)
	if err != nil {
		return nil, err
	}
	if exists {
		dat, err := os.ReadFile(b.fullPath)
		if err != nil {
			return nil, err
		}
		block, _ := big.NewInt(0).SetString(string(dat), 10)
		return block, nil
	}
	// Otherwise just return 0
	return big.NewInt(0), nil
}

func getFileName(chainName string) string {
	return fmt.Sprintf("%s.block", chainName)
}

// getHomePath returns the home directory joined with PathPostfix
func getDefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, PathPostfix), nil
}

func fileExists(fileName string) (bool, error) {
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (bs *Blockstore) FullPath() string {
	return bs.fullPath
}
