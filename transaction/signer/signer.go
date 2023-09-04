package signer

import (
	"crypto/ecdsa"

	secp256k1 "github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/common"
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

func NewKeypairFromPrivateKey(priv string) (*Keypair, error) {
	pk, err := secp256k1.HexToECDSA(priv)
	if err != nil {
		return nil, err
	}

	return &Keypair{
		public:  pk.Public().(*ecdsa.PublicKey),
		private: pk,
	}, nil
}

// CommonAddress returns the Ethereum address in the common.Address Format
func (kp *Keypair) CommonAddress() common.Address {
	return secp256k1.PubkeyToAddress(*kp.public)
}

// Sign calculates an ECDSA signature.
// The produced signature is in the [R || S || V] format where V is 0 or 1.
func (kp *Keypair) Sign(digestHash []byte) ([]byte, error) {
	return secp256k1.Sign(digestHash, kp.private)
}
