package core

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"go.etcd.io/bbolt"

	"my-blockchain/wallet"
)

const blocksBucket = "blocks"
const lastHashKey = "l"

const dbLockTimeout = 2 * time.Second

func openDB(nodeID string) (*bbolt.DB, error) {
	return bbolt.Open(nodeDBFile(nodeID), 0o600, &bbolt.Options{Timeout: dbLockTimeout})
}

func openDBReadOnly(nodeID string) (*bbolt.DB, error) {
	return bbolt.Open(nodeDBFile(nodeID), 0o600, &bbolt.Options{Timeout: dbLockTimeout, ReadOnly: true})
}

func nodeDBFile(nodeID string) string {
	if nodeID == "" {
		nodeID = "3000"
	}
	return fmt.Sprintf("blockchain_%s.db", nodeID)
}

type Blockchain struct {
	db  *bbolt.DB
	tip []byte
}

func NewGenesisBlock(coinbase *Transaction) *Block {
	genesis := &Block{
		Timestamp:     0,
		Transactions:  []*Transaction{coinbase},
		PrevBlockHash: []byte{},
		Hash:          nil,
		Nonce:         0,
		MerkleRoot:    nil,
	}
	genesis.MerkleRoot = genesis.HashTransactions()
	pow := NewProofOfWork(genesis)
	nonce, hash := pow.Run()
	genesis.Nonce = nonce
	genesis.Hash = hash
	return genesis
}

func dbExists(nodeID string) bool {
	_, err := os.Stat(nodeDBFile(nodeID))
	return err == nil
}

// DBExists reports whether the blockchain database file exists.
func DBExists(nodeID string) bool {
	return dbExists(nodeID)
}

// CreateBlockchain initializes a brand-new blockchain database.
func CreateBlockchain(address string) *Blockchain {
	if !wallet.ValidateAddress(address) {
		log.Panic("invalid address")
	}
	return CreateBlockchainForNode(address, os.Getenv("NODE_ID"))
}

func CreateBlockchainForNode(address string, nodeID string) *Blockchain {
	if !wallet.ValidateAddress(address) {
		log.Panic("invalid address")
	}
	if dbExists(nodeID) {
		log.Panic("blockchain database already exists")
	}

	db, err := openDB(nodeID)
	if err != nil {
		if errors.Is(err, bbolt.ErrTimeout) {
			log.Panicf("failed to open blockchain DB %q: timeout (if a node is running with the same NODE_ID, stop it and retry)", nodeDBFile(nodeID))
		}
		log.Panic(err)
	}

	var tip []byte
	err = db.Update(func(tx *bbolt.Tx) error {
		b, createErr := tx.CreateBucket([]byte(blocksBucket))
		if createErr != nil {
			return createErr
		}

		coinbase := CoinbaseTx(address, "Genesis")
		genesis := NewGenesisBlock(coinbase)
		if putErr := b.Put(genesis.Hash, genesis.Serialize()); putErr != nil {
			return putErr
		}
		if putErr := b.Put([]byte(lastHashKey), genesis.Hash); putErr != nil {
			return putErr
		}
		tip = genesis.Hash
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return &Blockchain{db: db, tip: tip}
}

// OpenBlockchain opens an existing blockchain database.
func OpenBlockchain() *Blockchain {
	return OpenBlockchainForNode(os.Getenv("NODE_ID"))
}

func OpenBlockchainForNode(nodeID string) *Blockchain {
	if !dbExists(nodeID) {
		log.Panic("no existing blockchain database found; run createblockchain first")
	}

	db, err := openDB(nodeID)
	if err != nil {
		if errors.Is(err, bbolt.ErrTimeout) {
			log.Panicf("failed to open blockchain DB %q: timeout (if a node is running with the same NODE_ID, stop it and retry)", nodeDBFile(nodeID))
		}
		log.Panic(err)
	}

	var tip []byte
	err = db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			log.Panic("blockchain database is missing blocks bucket")
		}
		tip = b.Get([]byte(lastHashKey))
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return &Blockchain{db: db, tip: tip}
}

// OpenBlockchainReadOnlyForNode opens an existing blockchain database in read-only mode.
// This allows commands like printchain/getbalance to run while a node process is running.
func OpenBlockchainReadOnlyForNode(nodeID string) *Blockchain {
	if !dbExists(nodeID) {
		log.Panic("no existing blockchain database found; run createblockchain first")
	}

	db, err := openDBReadOnly(nodeID)
	if err != nil {
		if errors.Is(err, bbolt.ErrTimeout) {
			log.Panicf("failed to open blockchain DB %q: timeout (if a node is running with the same NODE_ID, stop it and retry)", nodeDBFile(nodeID))
		}
		log.Panic(err)
	}

	var tip []byte
	err = db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			log.Panic("blockchain database is missing blocks bucket")
		}
		tip = b.Get([]byte(lastHashKey))
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return &Blockchain{db: db, tip: tip}
}

// InitBlockchainForNode opens the DB for a node and ensures the bucket exists.
// It does NOT create a genesis block. Used by networking nodes that will sync from peers.
func InitBlockchainForNode(nodeID string) *Blockchain {
	db, err := openDB(nodeID)
	if err != nil {
		if errors.Is(err, bbolt.ErrTimeout) {
			log.Panicf("failed to open blockchain DB %q: timeout (if a node is running with the same NODE_ID, stop it and retry)", nodeDBFile(nodeID))
		}
		log.Panic(err)
	}

	var tip []byte
	err = db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			var createErr error
			b, createErr = tx.CreateBucket([]byte(blocksBucket))
			if createErr != nil {
				return createErr
			}
		}
		tip = b.Get([]byte(lastHashKey))
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return &Blockchain{db: db, tip: tip}
}

func (bc *Blockchain) Close() error {
	if bc.db == nil {
		return nil
	}
	return bc.db.Close()
}

func (bc *Blockchain) Tip() []byte {
	return bc.tip
}

func (bc *Blockchain) AddBlock(transactions []*Transaction) []byte {
	for _, tx := range transactions {
		if !bc.VerifyTransaction(tx) {
			log.Panic("invalid transaction")
		}
	}

	var lastHash []byte

	err := bc.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte(lastHashKey))
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	newBlock := NewBlock(transactions, lastHash)

	err = bc.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if putErr := b.Put(newBlock.Hash, newBlock.Serialize()); putErr != nil {
			return putErr
		}
		if putErr := b.Put([]byte(lastHashKey), newBlock.Hash); putErr != nil {
			return putErr
		}
		bc.tip = newBlock.Hash
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	return newBlock.Hash
}

func (bc *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	it := bc.Iterator()
	for {
		block := it.Next()
		for _, tx := range block.Transactions {
			if bytes.Equal(tx.ID, ID) {
				return *tx, nil
			}
		}
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return Transaction{}, errors.New("transaction not found")
}

func (bc *Blockchain) SignTransaction(tx *Transaction, privKey *ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)
	for _, vin := range tx.Vin {
		prevTx, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTx.ID)] = prevTx
	}
	tx.Sign(privKey, prevTXs)
}

func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}
	prevTXs := make(map[string]Transaction)
	for _, vin := range tx.Vin {
		prevTx, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTx.ID)] = prevTx
	}
	return tx.Verify(prevTXs)
}
