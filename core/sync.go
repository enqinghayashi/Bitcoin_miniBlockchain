package core

import (
	"bytes"
	"errors"
	"log"

	"go.etcd.io/bbolt"
)

func (bc *Blockchain) BestHeight() int {
	if bc.tip == nil {
		return 0
	}
	it := bc.Iterator()
	height := 0
	for {
		block := it.Next()
		if block == nil {
			break
		}
		height++
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return height
}

// GetBlockHashes returns all known block hashes in chain order (genesis -> tip).
func (bc *Blockchain) GetBlockHashes() [][]byte {
	if bc.tip == nil {
		return nil
	}
	it := bc.Iterator()
	var hashes [][]byte
	for {
		block := it.Next()
		if block == nil {
			break
		}
		hashes = append(hashes, append([]byte(nil), block.Hash...))
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	// reverse (currently tip -> genesis)
	for i, j := 0, len(hashes)-1; i < j; i, j = i+1, j-1 {
		hashes[i], hashes[j] = hashes[j], hashes[i]
	}
	return hashes
}

func (bc *Blockchain) HasBlock(hash []byte) bool {
	found := false
	_ = bc.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			return nil
		}
		found = b.Get(hash) != nil
		return nil
	})
	return found
}

func (bc *Blockchain) GetBlock(hash []byte) ([]byte, error) {
	var data []byte
	err := bc.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			return errors.New("missing blocks bucket")
		}
		v := b.Get(hash)
		if v == nil {
			return errors.New("block not found")
		}
		data = append([]byte(nil), v...)
		return nil
	})
	return data, err
}

// PutBlock stores a serialized block in the DB. It updates the tip if the block extends the current tip.
func (bc *Blockchain) PutBlock(blockData []byte) {
	block := DeserializeBlock(blockData)

	err := bc.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			var createErr error
			b, createErr = tx.CreateBucket([]byte(blocksBucket))
			if createErr != nil {
				return createErr
			}
		}

		if existing := b.Get(block.Hash); existing == nil {
			if err := b.Put(block.Hash, blockData); err != nil {
				return err
			}
		}

		currentTip := b.Get([]byte(lastHashKey))
		// Empty chain: accept first block as tip.
		if currentTip == nil || len(currentTip) == 0 {
			if err := b.Put([]byte(lastHashKey), block.Hash); err != nil {
				return err
			}
			bc.tip = block.Hash
			return nil
		}

		// Simple linear-chain rule: update tip only if it directly extends the current tip.
		if bytes.Equal(block.PrevBlockHash, currentTip) {
			if err := b.Put([]byte(lastHashKey), block.Hash); err != nil {
				return err
			}
			bc.tip = block.Hash
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}
