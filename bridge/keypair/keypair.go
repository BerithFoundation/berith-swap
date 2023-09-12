package keypair

import (
	"crypto/ecdsa"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	secp256k1 "github.com/ethereum/go-ethereum/crypto"
)

type Signer interface {
	CommonAddress() common.Address
	Sign(digestHash []byte) ([]byte, error)
}

type Keypair struct {
	public  *ecdsa.PublicKey
	private *ecdsa.PrivateKey
}

func GenerateKeyPair(addr, path, password string) (*Keypair, error) {
	if password == "" {
		return nil, fmt.Errorf("password is empty")
	}
	pattern := fmt.Sprintf("%s/*%s", path, strings.ToLower(addr[2:]))
	keys, err := filepath.Glob(pattern)
	if err != nil || len(keys) == 0 {
		return nil, fmt.Errorf("no key files matching given keyword:%s", addr)
	}

	kp, err := ReadFromFileAndDecrypt(keys[0], password)
	if err != nil {
		return nil, err
	}

	return kp, nil
}

func NewKeypairFromPrivateKey(priv *ecdsa.PrivateKey) *Keypair {
	return &Keypair{
		public:  priv.Public().(*ecdsa.PublicKey),
		private: priv,
	}
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
