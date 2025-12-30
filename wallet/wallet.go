package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"math/big"

	"golang.org/x/crypto/ripemd160"
)

const (
	addressVersion     = byte(0x00)
	addressChecksumLen = 4
	privateKeyByteLen  = 32 // P-256 scalar size
)

type Wallet struct {
	// We store the private scalar D as bytes to make persistence easy.
	PrivateKey []byte
	// Uncompressed public key bytes (elliptic.Marshal)
	PublicKey []byte
}

func NewWallet() *Wallet {
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	pubKey := elliptic.Marshal(elliptic.P256(), private.PublicKey.X, private.PublicKey.Y)
	privKeyBytes := private.D.Bytes()
	if len(privKeyBytes) < privateKeyByteLen {
		padded := make([]byte, privateKeyByteLen)
		copy(padded[privateKeyByteLen-len(privKeyBytes):], privKeyBytes)
		privKeyBytes = padded
	}

	return &Wallet{PrivateKey: privKeyBytes, PublicKey: pubKey}
}

func (w *Wallet) PrivateECDSA() *ecdsa.PrivateKey {
	curve := elliptic.P256()
	d := new(big.Int).SetBytes(w.PrivateKey)
	x, y := curve.ScalarBaseMult(w.PrivateKey)
	return &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{Curve: curve, X: x, Y: y},
		D:         d,
	}
}

// HashPubKey performs SHA256 then RIPEMD160 (Bitcoin-style).
func HashPubKey(pubKey []byte) []byte {
	publicSHA := sha256.Sum256(pubKey)
	ri := ripemd160.New()
	_, _ = ri.Write(publicSHA[:])
	return ri.Sum(nil)
}

func (w *Wallet) GetAddress() []byte {
	pubKeyHash := HashPubKey(w.PublicKey)
	versionedPayload := append([]byte{addressVersion}, pubKeyHash...)
	checksum := checksum(versionedPayload)
	fullPayload := append(versionedPayload, checksum...)
	return Base58Encode(fullPayload)
}

func ValidateAddress(address string) bool {
	decoded := Base58Decode([]byte(address))
	if decoded == nil || len(decoded) < 1+addressChecksumLen {
		return false
	}
	payload := decoded[:len(decoded)-addressChecksumLen]
	actualChecksum := decoded[len(decoded)-addressChecksumLen:]
	expectedChecksum := checksum(payload)
	return bytes.Equal(actualChecksum, expectedChecksum)
}

func PubKeyHashFromAddress(address string) []byte {
	decoded := Base58Decode([]byte(address))
	if decoded == nil || len(decoded) < 1+addressChecksumLen {
		return nil
	}
	// version (1 byte) | pubKeyHash (20 bytes) | checksum (4 bytes)
	pubKeyHash := decoded[1 : len(decoded)-addressChecksumLen]
	return pubKeyHash
}

func checksum(payload []byte) []byte {
	first := sha256.Sum256(payload)
	second := sha256.Sum256(first[:])
	return second[:addressChecksumLen]
}
