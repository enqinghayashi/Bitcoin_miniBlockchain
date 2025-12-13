package network

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"my-blockchain/core"
	"my-blockchain/wallet"
)

const protocolVersion = 1

// For the milestone, we keep a simple fixed set of peers on localhost.
var knownNodes = []string{"localhost:3000", "localhost:3001", "localhost:3002"}

var nodeAddress string
var blocksInTransit [][]byte

var minerAddr string

type Message struct {
	Command string
	Payload []byte
}

type Version struct {
	Version    int
	BestHeight int
	AddrFrom   string
}

type GetBlocks struct {
	AddrFrom string
}

type Inv struct {
	AddrFrom string
	Type     string
	Items    [][]byte
}

type GetData struct {
	AddrFrom string
	Type     string
	ID       []byte
}

type BlockData struct {
	AddrFrom string
	Block    []byte
}

// BalanceRequest asks the node to compute the UTXO balance for an address.
type BalanceRequest struct {
	AddrFrom string
	Address  string
}

type BalanceResponse struct {
	OK      bool
	Message string
	Balance int
}

// ChainRequest asks the node to return a printable view of the current chain.
type ChainRequest struct {
	AddrFrom string
}

type ChainBlock struct {
	Index     int
	Timestamp int64
	PrevHash  []byte
	Hash      []byte
	Nonce     int
	Merkle    []byte
	TxIDs     [][]byte
}

type ChainResponse struct {
	OK      bool
	Message string
	Blocks  []ChainBlock
}

// TxRequest is an RPC-style request asking the node to construct/sign a transaction
// (using local wallets.dat), mine it into a block, and persist/broadcast the block.
type TxRequest struct {
	AddrFrom string
	From     string
	To       string
	Amount   int
}

// Result is a generic request/response payload.
type Result struct {
	OK      bool
	Message string
}

