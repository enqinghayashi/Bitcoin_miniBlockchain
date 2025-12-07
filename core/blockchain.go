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

