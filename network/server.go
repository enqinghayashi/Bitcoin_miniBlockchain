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

func StartServer(nodeID string, minerAddress string) {
	minerAddr = minerAddress

	nodeAddress = fmt.Sprintf("localhost:%s", nodeID)
	bc := core.InitBlockchainForNode(nodeID)
	defer func() { _ = bc.Close() }()

	ln, err := net.Listen("tcp", nodeAddress)
	if err != nil {
		log.Panic(err)
	}
	defer func() { _ = ln.Close() }()

	if minerAddr != "" {
		log.Printf("Node %s listening (db=%s, miner=%s)\n", nodeAddress, "blockchain_"+nodeID+".db", minerAddr)
	} else {
		log.Printf("Node %s listening (db=%s)\n", nodeAddress, "blockchain_"+nodeID+".db")
	}

	// If we're not the bootstrap node, announce ourselves.
	if nodeAddress != knownNodes[0] {
		go sendVersion(knownNodes[0], bc)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn, bc)
	}
}

func currentNodeID() string {
	return strings.TrimPrefix(nodeAddress, "localhost:")
}

func handleConnection(conn net.Conn, bc *core.Blockchain) {
	defer func() { _ = conn.Close() }()
	_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	dec := gob.NewDecoder(conn)
	var msg Message
	if err := dec.Decode(&msg); err != nil {
		return
	}

	switch msg.Command {
	case "version":
		handleVersion(msg.Payload, bc)
	case "getblocks":
		handleGetBlocks(msg.Payload, bc)
	case "inv":
		handleInv(msg.Payload, bc)
	case "getdata":
		handleGetData(msg.Payload, bc)
	case "block":
		handleBlock(msg.Payload, bc)
	case "sendtx":
		handleSendTx(conn, msg.Payload, bc)
	case "getbalance":
		handleGetBalance(conn, msg.Payload, bc)
	case "getchain":
		handleGetChain(conn, msg.Payload, bc)
	default:
		// ignore unknown
	}
}

