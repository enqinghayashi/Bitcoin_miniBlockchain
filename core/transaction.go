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

func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Coinbase to %s", to)
	}

	txin := TxInput{Txid: []byte{}, Vout: -1, Signature: nil, PubKey: []byte(data)}
	txout := *NewTxOutput(subsidy, to)

	tx := &Transaction{ID: nil, Vin: []TxInput{txin}, Vout: []TxOutput{txout}}
	tx.ID = tx.Hash()
	return tx
}

func (tx *Transaction) Serialize() []byte {
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	if err := enc.Encode(tx); err != nil {
		log.Panic(err)
	}
	return encoded.Bytes()
}

func (tx *Transaction) Hash() []byte {
	txCopy := *tx
	txCopy.ID = nil

	hash := sha256.Sum256(txCopy.Serialize())
	return hash[:]
}

func (tx *Transaction) TrimmedCopy() Transaction {
	inputs := make([]TxInput, 0, len(tx.Vin))
	for _, vin := range tx.Vin {
		inputs = append(inputs, TxInput{Txid: vin.Txid, Vout: vin.Vout, Signature: nil, PubKey: nil})
	}
	outputs := make([]TxOutput, 0, len(tx.Vout))
	for _, vout := range tx.Vout {
		outputs = append(outputs, TxOutput{Value: vout.Value, PubKeyHash: vout.PubKeyHash})
	}
	return Transaction{ID: tx.ID, Vin: inputs, Vout: outputs}
}

