package core

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"

	"my-blockchain/wallet"
)

const subsidy = 10

type Transaction struct {
	ID   []byte
	Vin  []TxInput
	Vout []TxOutput
}

type TxInput struct {
	Txid      []byte
	Vout      int
	Signature []byte
	PubKey    []byte
}

type TxOutput struct {
	Value      int
	PubKeyHash []byte
}

func (in *TxInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := wallet.HashPubKey(in.PubKey)
	return bytes.Equal(lockingHash, pubKeyHash)
}

func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}

func (out *TxOutput) Lock(address string) error {
	pubKeyHash := wallet.PubKeyHashFromAddress(address)
	if pubKeyHash == nil {
		return errors.New("invalid address")
	}
	out.PubKeyHash = pubKeyHash
	return nil
}

func NewTxOutput(value int, address string) *TxOutput {
	out := &TxOutput{Value: value}
	if err := out.Lock(address); err != nil {
		log.Panic(err)
	}
	return out
}

