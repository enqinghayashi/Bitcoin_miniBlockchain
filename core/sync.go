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

