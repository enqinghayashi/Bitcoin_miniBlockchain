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

func sendReply(conn net.Conn, msg Message) {
	enc := gob.NewEncoder(conn)
	_ = enc.Encode(msg)
}

func encodePayload(v any) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		log.Panic(err)
	}
	return buf.Bytes()
}

func decodePayload(data []byte, out any) {
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(out); err != nil {
		log.Panic(err)
	}
}

func sendData(addr string, msg Message) {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	enc := gob.NewEncoder(conn)
	_ = enc.Encode(msg)
}

// sendRequest sends a message and waits for a single reply message.
func sendRequest(addr string, msg Message) (*Message, error) {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	enc := gob.NewEncoder(conn)
	if err := enc.Encode(msg); err != nil {
		return nil, err
	}

	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	dec := gob.NewDecoder(conn)
	var reply Message
	if err := dec.Decode(&reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// SendTxRequest asks the running node at localhost:<nodeID> to construct/sign/mine a transaction.
// This avoids opening BoltDB from the CLI process while startnode owns the DB.
func SendTxRequest(nodeID string, from string, to string, amount int) (string, error) {
	addr := fmt.Sprintf("localhost:%s", nodeID)
	payload := TxRequest{AddrFrom: addr, From: from, To: to, Amount: amount}
	reply, err := sendRequest(addr, Message{Command: "sendtx", Payload: encodePayload(payload)})
	if err != nil {
		return "", err
	}
	if reply.Command != "result" {
		return "", fmt.Errorf("unexpected reply: %s", reply.Command)
	}
	var res Result
	decodePayload(reply.Payload, &res)
	if !res.OK {
		return "", fmt.Errorf(res.Message)
	}
	return res.Message, nil
}

// GetBalanceRequest asks the running node at localhost:<nodeID> for an address balance.
func GetBalanceRequest(nodeID string, address string) (int, error) {
	addr := fmt.Sprintf("localhost:%s", nodeID)
	payload := BalanceRequest{AddrFrom: addr, Address: address}
	reply, err := sendRequest(addr, Message{Command: "getbalance", Payload: encodePayload(payload)})
	if err != nil {
		return 0, err
	}
	if reply.Command != "balance" {
		return 0, fmt.Errorf("unexpected reply: %s", reply.Command)
	}
	var res BalanceResponse
	decodePayload(reply.Payload, &res)
	if !res.OK {
		return 0, fmt.Errorf(res.Message)
	}
	return res.Balance, nil
}

// GetChainRequest asks the running node at localhost:<nodeID> for a chain snapshot to print.
func GetChainRequest(nodeID string) ([]ChainBlock, string, error) {
	addr := fmt.Sprintf("localhost:%s", nodeID)
	payload := ChainRequest{AddrFrom: addr}
	reply, err := sendRequest(addr, Message{Command: "getchain", Payload: encodePayload(payload)})
	if err != nil {
		return nil, "", err
	}
	if reply.Command != "chain" {
		return nil, "", fmt.Errorf("unexpected reply: %s", reply.Command)
	}
	var res ChainResponse
	decodePayload(reply.Payload, &res)
	if !res.OK {
		return nil, res.Message, fmt.Errorf(res.Message)
	}
	return res.Blocks, res.Message, nil
}

func sendVersion(addr string, bc *core.Blockchain) {
	payload := Version{Version: protocolVersion, BestHeight: bc.BestHeight(), AddrFrom: nodeAddress}
	sendData(addr, Message{Command: "version", Payload: encodePayload(payload)})
}

func sendGetBlocks(addr string) {
	payload := GetBlocks{AddrFrom: nodeAddress}
	sendData(addr, Message{Command: "getblocks", Payload: encodePayload(payload)})
}

func sendInv(addr string, kind string, items [][]byte) {
	payload := Inv{AddrFrom: nodeAddress, Type: kind, Items: items}
	sendData(addr, Message{Command: "inv", Payload: encodePayload(payload)})
}

func sendGetData(addr string, kind string, id []byte) {
	payload := GetData{AddrFrom: nodeAddress, Type: kind, ID: id}
	sendData(addr, Message{Command: "getdata", Payload: encodePayload(payload)})
}

func sendBlock(addr string, blockBytes []byte) {
	payload := BlockData{AddrFrom: nodeAddress, Block: blockBytes}
	sendData(addr, Message{Command: "block", Payload: encodePayload(payload)})
}

func handleVersion(payloadBytes []byte, bc *core.Blockchain) {
	var payload Version
	decodePayload(payloadBytes, &payload)

	myBestHeight := bc.BestHeight()
	if myBestHeight < payload.BestHeight {
		sendGetBlocks(payload.AddrFrom)
	} else if myBestHeight > payload.BestHeight {
		sendVersion(payload.AddrFrom, bc)
	}
}

func handleGetBlocks(payloadBytes []byte, bc *core.Blockchain) {
	var payload GetBlocks
	decodePayload(payloadBytes, &payload)

	hashes := bc.GetBlockHashes()
	sendInv(payload.AddrFrom, "block", hashes)
}

func handleInv(payloadBytes []byte, bc *core.Blockchain) {
	var payload Inv
	decodePayload(payloadBytes, &payload)
	if payload.Type != "block" {
		return
	}

	// Request blocks we don't have, in the order provided.
	blocksInTransit = nil
	for _, h := range payload.Items {
		if !bc.HasBlock(h) {
			blocksInTransit = append(blocksInTransit, h)
		}
	}
	if len(blocksInTransit) == 0 {
		return
	}

	// Request the first missing block.
	request := blocksInTransit[0]
	blocksInTransit = blocksInTransit[1:]
	sendGetData(payload.AddrFrom, "block", request)
}

func handleGetData(payloadBytes []byte, bc *core.Blockchain) {
	var payload GetData
	decodePayload(payloadBytes, &payload)
	if payload.Type != "block" {
		return
	}

	blockBytes, err := bc.GetBlock(payload.ID)
	if err != nil {
		return
	}
	sendBlock(payload.AddrFrom, blockBytes)
}

func handleBlock(payloadBytes []byte, bc *core.Blockchain) {
	var payload BlockData
	decodePayload(payloadBytes, &payload)

	bc.PutBlock(payload.Block)

	if len(blocksInTransit) > 0 {
		next := blocksInTransit[0]
		blocksInTransit = blocksInTransit[1:]
		sendGetData(payload.AddrFrom, "block", next)
		return
	}

	// After syncing, announce our version to the bootstrap so it can respond if needed.
	if nodeAddress != knownNodes[0] {
		sendVersion(knownNodes[0], bc)
	}
}

