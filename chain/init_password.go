package chain

import (
	"berith-swap/keypair"
	"fmt"
	"path/filepath"
)

func GenerateKeyPair(addr, path, password string) (*keypair.Keypair, error) {
	if password == "" {
		return nil, fmt.Errorf("password is empty")
	}
	keys, err := filepath.Glob(fmt.Sprintf("*%s", addr[2:]))
	if err != nil {
		return nil, fmt.Errorf("no key files matching given keyword:%s", addr)
	}
	path = fmt.Sprintf("%s/%s.key", path, keys[0])

	var pswd []byte = []byte(password)

	kp, err := keypair.ReadFromFileAndDecrypt(path, pswd)
	if err != nil {
		return nil, err
	}

	return kp, nil
}
