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

