package core

import (
	"bytes"
	"encoding/hex"
	"log"

	"my-blockchain/wallet"
)

func (bc *Blockchain) FindUnspentTransactions(pubKeyHash []byte) []Transaction {
	var unspentTXs []Transaction
	spentTXOs := make(map[string][]int)

	it := bc.Iterator()
	for {
		block := it.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Vout {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}

				if out.IsLockedWithKey(pubKeyHash) {
					unspentTXs = append(unspentTXs, *tx)
				}
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.Vin {
					if in.UsesKey(pubKeyHash) {
						inTxID := hex.EncodeToString(in.Txid)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)
					}
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return unspentTXs
}

func (bc *Blockchain) FindUTXO(pubKeyHash []byte) []TxOutput {
	var UTXOs []TxOutput
	unspentTxs := bc.FindUnspentTransactions(pubKeyHash)
	for _, tx := range unspentTxs {
		for _, out := range tx.Vout {
			if out.IsLockedWithKey(pubKeyHash) {
				UTXOs = append(UTXOs, out)
			}
		}
	}
	return UTXOs
}

func (bc *Blockchain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTxs := bc.FindUnspentTransactions(pubKeyHash)
	accumulated := 0

Work:
	for _, tx := range unspentTxs {
		txID := hex.EncodeToString(tx.ID)
		for outIdx, out := range tx.Vout {
			if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
				accumulated += out.Value
				unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)
				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOutputs
}

func NewUTXOTransaction(from, to string, amount int, bc *Blockchain, ws *wallet.Wallets) *Transaction {
	if !wallet.ValidateAddress(from) || !wallet.ValidateAddress(to) {
		log.Panic("invalid from/to address")
	}

	w, ok := ws.GetWallet(from)
	if !ok {
		log.Panic("sender wallet not found; createwallet first")
	}

	fromPubKeyHash := wallet.PubKeyHashFromAddress(from)
	toPubKeyHash := wallet.PubKeyHashFromAddress(to)
	if fromPubKeyHash == nil || toPubKeyHash == nil {
		log.Panic("invalid address")
	}

	acc, validOutputs := bc.FindSpendableOutputs(fromPubKeyHash, amount)
	if acc < amount {
		log.Panic("not enough funds")
	}

	var inputs []TxInput
	var outputs []TxOutput

	for txidStr, outs := range validOutputs {
		txIDBytes, err := hex.DecodeString(txidStr)
		if err != nil {
			log.Panic(err)
		}
		for _, outIdx := range outs {
			input := TxInput{Txid: txIDBytes, Vout: outIdx, Signature: nil, PubKey: w.PublicKey}
			inputs = append(inputs, input)
		}
	}

	// outputs
	outputs = append(outputs, TxOutput{Value: amount, PubKeyHash: append([]byte(nil), toPubKeyHash...)})
	if acc > amount {
		outputs = append(outputs, TxOutput{Value: acc - amount, PubKeyHash: append([]byte(nil), fromPubKeyHash...)})
	}

	tx := &Transaction{ID: nil, Vin: inputs, Vout: outputs}
	tx.ID = tx.Hash()

	bc.SignTransaction(tx, w.PrivateECDSA())

	// Basic sanity: ensure each input matches the sender key.
	for _, vin := range tx.Vin {
		if !bytes.Equal(wallet.HashPubKey(vin.PubKey), fromPubKeyHash) {
			log.Panic("input pubkey does not match sender")
		}
	}

	return tx
}
