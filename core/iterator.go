package core

import (
	"log"

	"go.etcd.io/bbolt"
)

type BlockchainIterator struct {
	currentHash []byte
	db          *bbolt.DB
}

func (bc *Blockchain) Iterator() *BlockchainIterator {
	return &BlockchainIterator{currentHash: bc.tip, db: bc.db}
}

func (it *BlockchainIterator) Next() *Block {
	if len(it.currentHash) == 0 {
		return nil
	}
	var block *Block

	err := it.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encoded := b.Get(it.currentHash)
		if encoded == nil {
			block = nil
			return nil
		}
		block = DeserializeBlock(encoded)
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	if block == nil {
		return nil
	}

	it.currentHash = block.PrevBlockHash
	return block
}
