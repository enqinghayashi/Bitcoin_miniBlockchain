package cli

import (
	"flag"
	"fmt"
	"os"

	"my-blockchain/core"
	"my-blockchain/network"
	"my-blockchain/wallet"
)

type CLI struct{}

func nodeID() string {
	id := os.Getenv("NODE_ID")
	if id == "" {
		id = "3000"
	}
	return id
}

func (c *CLI) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  createwallet")
	fmt.Println("  listaddresses")
	fmt.Println("  createblockchain -address YOUR_ADDRESS")
	fmt.Println("  printchain")
	fmt.Println("  getbalance -address YOUR_ADDRESS")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT")
	fmt.Println("  startnode -miner MINER_ADDRESS(optional)")
}

func (c *CLI) validateArgs() {
	if len(os.Args) < 2 {
		c.printUsage()
		os.Exit(1)
	}
}

func (c *CLI) createBlockchain(address string) {
	if core.DBExists(nodeID()) {
		fmt.Printf("Blockchain already exists. Delete %s to recreate.\n", "blockchain_"+nodeID()+".db")
		return
	}
	bc := core.CreateBlockchainForNode(address, nodeID())
	defer func() { _ = bc.Close() }()
	fmt.Println("Done! Created a new blockchain.")
}

func (c *CLI) printChain() {
	// Ask the running node to print chain state.
	blocks, msg, err := network.GetChainRequest(nodeID())
	if err == nil {
		if msg != "" {
			fmt.Println(msg)
		}
		for _, b := range blocks {
			fmt.Printf("===== Block %d =====\n", b.Index)
			fmt.Printf("Timestamp: %d\n", b.Timestamp)
			fmt.Printf("Prev. hash: %x\n", b.PrevHash)
			fmt.Printf("Hash: %x\n", b.Hash)
			fmt.Printf("Nonce: %d\n", b.Nonce)
			fmt.Printf("Merkle: %x\n", b.Merkle)
			fmt.Printf("Tx count: %d\n", len(b.TxIDs))
			for _, txid := range b.TxIDs {
				fmt.Printf("  TxID: %x\n", txid)
			}
			fmt.Println()
		}
		return
	}

	// Fallback for offline/single-process usage.
	if !core.DBExists(nodeID()) {
		fmt.Println("No blockchain found. Run: createblockchain -address YOUR_ADDRESS")
		return
	}
	bc := core.OpenBlockchainReadOnlyForNode(nodeID())
	defer func() { _ = bc.Close() }()

	if len(bc.Tip()) == 0 {
		fmt.Printf("Blockchain DB exists for node %s, but it has no blocks yet.\n", nodeID())
		fmt.Println("If this is a networking node, run: startnode (and make sure node 3000 is running).")
		fmt.Println("If you want a standalone chain on this node, run: createblockchain -address YOUR_ADDRESS (after deleting the DB).")
		return
	}

	it := bc.Iterator()
	index := 0
	for {
		block := it.Next()
		if block == nil {
			break
		}
		fmt.Printf("===== Block %d =====\n", index)
		fmt.Printf("Timestamp: %d\n", block.Timestamp)
		fmt.Printf("Prev. hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Hash: %x\n", block.Hash)
		fmt.Printf("Nonce: %d\n", block.Nonce)
		fmt.Printf("Merkle: %x\n", block.MerkleRoot)
		fmt.Printf("Tx count: %d\n", len(block.Transactions))
		for _, tx := range block.Transactions {
			fmt.Printf("  TxID: %x\n", tx.ID)
		}
		fmt.Println()
		index++

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
}

