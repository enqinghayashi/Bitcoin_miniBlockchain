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

func (tx *Transaction) Sign(privKey *ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}

	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()
	for inID, vin := range txCopy.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PubKey = nil

		sig, err := ecdsa.SignASN1(rand.Reader, privKey, txCopy.ID)
		if err != nil {
			log.Panic(err)
		}
		tx.Vin[inID].Signature = sig
	}
}

func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID, vin := range tx.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PubKey = nil

		x, y := elliptic.Unmarshal(curve, vin.PubKey)
		if x == nil {
			return false
		}
		pubKey := ecdsa.PublicKey{Curve: curve, X: x, Y: y}
		if !ecdsa.VerifyASN1(&pubKey, txCopy.ID, vin.Signature) {
			return false
		}
	}

	return true
}

func (tx *Transaction) String() string {
	var lines []string
	lines = append(lines, fmt.Sprintf("--- Transaction %x", tx.ID))

	for i, input := range tx.Vin {
		lines = append(lines, fmt.Sprintf("  Input %d:", i))
		lines = append(lines, fmt.Sprintf("    TXID: %x", input.Txid))
		lines = append(lines, fmt.Sprintf("    Out:  %d", input.Vout))
		lines = append(lines, fmt.Sprintf("    Sig:  %x", input.Signature))
		lines = append(lines, fmt.Sprintf("    Pub:  %x", input.PubKey))
	}

	for i, output := range tx.Vout {
		lines = append(lines, fmt.Sprintf("  Output %d:", i))
		lines = append(lines, fmt.Sprintf("    Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("    Script: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}
