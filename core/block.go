package core

import (
	"bytes"
	"encoding/gob"
	"log"
	"time"
)

type Block struct {
	Timestamp     int64
	Transactions  []*Transaction
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int
	MerkleRoot    []byte
}

func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	if err := encoder.Encode(b); err != nil {
		log.Panic(err)
	}
	return result.Bytes()
}

func DeserializeBlock(data []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&block); err != nil {
		log.Panic(err)
	}
	return &block
}

func (b *Block) HashTransactions() []byte {
	txHashes := make([][]byte, 0, len(b.Transactions))
	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.ID)
	}
	tree := NewMerkleTree(txHashes)
	return tree.RootNode.Data
}

func NewBlock(transactions []*Transaction, prevBlockHash []byte) *Block {
	block := &Block{
		Timestamp:     time.Now().Unix(),
		Transactions:  transactions,
		PrevBlockHash: prevBlockHash,
		Hash:          nil,
		Nonce:         0,
		MerkleRoot:    nil,
	}
	block.MerkleRoot = block.HashTransactions()
	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()
	block.Nonce = nonce
	block.Hash = hash
	return block
}
