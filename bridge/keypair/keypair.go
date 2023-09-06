package keypair

import (
	"crypto/ecdsa"
	"fmt"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	secp256k1 "github.com/ethereum/go-ethereum/crypto"
)

type Signer interface {
	CommonAddress() common.Address
	Sign(digestHash []byte) ([]byte, error)
}

const PrivateKeyLength = 32

type Keypair struct {
	public  *ecdsa.PublicKey
	private *ecdsa.PrivateKey
}

func GenerateKeyPair(addr, path, password string) (*Keypair, error) {
	if password == "" {
		return nil, fmt.Errorf("password is empty")
	}
	keys, err := filepath.Glob(fmt.Sprintf("*%s", addr[2:]))
	if err != nil {
		return nil, fmt.Errorf("no key files matching given keyword:%s", addr)
	}
	path = fmt.Sprintf("%s/%s.key", path, keys[0])

	var pswd []byte = []byte(password)

	kp, err := ReadFromFileAndDecrypt(path, pswd)
	if err != nil {
		return nil, err
	}

	return kp, nil
}

func NewKeypairFromPrivateKey(priv []byte) (*Keypair, error) {
	pk, err := secp256k1.ToECDSA(priv)
	if err != nil {
		return nil, err
	}

	return &Keypair{
		public:  pk.Public().(*ecdsa.PublicKey),
		private: pk,
	}, nil
}

// Encode dumps the private key as bytes
func (kp *Keypair) Encode() []byte {
	return secp256k1.FromECDSA(kp.private)
}

// Decode initializes the keypair using the input
func (kp *Keypair) Decode(in []byte) error {
	key, err := secp256k1.ToECDSA(in)
	if err != nil {
		return err
	}

	kp.public = key.Public().(*ecdsa.PublicKey)
	kp.private = key

	return nil
}

// Address returns the Ethereum address format
func (kp *Keypair) Address() string {
	return secp256k1.PubkeyToAddress(*kp.public).String()
}

// CommonAddress returns the Ethereum address in the common.Address Format
func (kp *Keypair) CommonAddress() common.Address {
	return secp256k1.PubkeyToAddress(*kp.public)
}

// PublicKey returns the public key hex encoded
func (kp *Keypair) PublicKey() string {
	return hexutil.Encode(secp256k1.CompressPubkey(kp.public))
}

// PrivateKey returns the keypair's private key
func (kp *Keypair) PrivateKey() *ecdsa.PrivateKey {
	return kp.private
}

// Sign calculates an ECDSA signature.
// The produced signature is in the [R || S || V] format where V is 0 or 1.
func (kp *Keypair) Sign(digestHash []byte) ([]byte, error) {
	return secp256k1.Sign(digestHash, kp.private)
}
