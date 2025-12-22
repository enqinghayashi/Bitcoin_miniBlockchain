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

func (c *CLI) getBalance(address string) {
	if !wallet.ValidateAddress(address) {
		fmt.Println("Invalid address")
		return
	}

	// Ask the running node to compute balance.
	balance, err := network.GetBalanceRequest(nodeID(), address)
	if err == nil {
		fmt.Printf("Balance of '%s': %d\n", address, balance)
		return
	}

	// Fallback for offline/single-process usage.
	if !core.DBExists(nodeID()) {
		fmt.Println("No blockchain found. Run: createblockchain -address YOUR_ADDRESS")
		return
	}
	bc := core.OpenBlockchainReadOnlyForNode(nodeID())
	defer func() { _ = bc.Close() }()

	pubKeyHash := wallet.PubKeyHashFromAddress(address)
	UTXOs := bc.FindUTXO(pubKeyHash)
	balance = 0
	for _, out := range UTXOs {
		balance += out.Value
	}
	fmt.Printf("Balance of '%s': %d\n", address, balance)
}

func (c *CLI) send(from, to string, amount int) {
	if !wallet.ValidateAddress(from) || !wallet.ValidateAddress(to) {
		fmt.Println("Invalid from/to address")
		return
	}

	msg, err := network.SendTxRequest(nodeID(), from, to, amount)
	if err != nil {
		// Fallback for single-node/offline usage: mine locally if no server is running.
		fmt.Println("Send via running node failed:", err)
		fmt.Println("Falling back to local mining (startnode not required).")
		if !core.DBExists(nodeID()) {
			fmt.Println("No blockchain found. Run: createblockchain -address YOUR_ADDRESS")
			return
		}
		ws, werr := wallet.NewWallets()
		if werr != nil {
			fmt.Println("Failed to load wallets:", werr)
			return
		}
		bc := core.OpenBlockchainForNode(nodeID())
		defer func() { _ = bc.Close() }()
		tx := core.NewUTXOTransaction(from, to, amount, bc, ws)
		cb := core.CoinbaseTx(from, "")
		newTip := bc.AddBlock([]*core.Transaction{cb, tx})
		fmt.Println("Success! Transaction mined into a new block.")
		network.BroadcastNewBlock(nodeID(), newTip)
		return
	}
	fmt.Println(msg)
}

func (c *CLI) startNode(miner string) {
	if miner != "" && !wallet.ValidateAddress(miner) {
		fmt.Println("Invalid miner address")
		return
	}
	network.StartServer(nodeID(), miner)
}

func (c *CLI) createWallet() {
	ws, err := wallet.NewWallets()
	if err != nil {
		fmt.Println("Failed to load wallets:", err)
		return
	}
	address, err := ws.CreateWallet()
	if err != nil {
		fmt.Println("Failed to create wallet:", err)
		return
	}
	fmt.Println("New address:", address)
}

func (c *CLI) listAddresses() {
	ws, err := wallet.NewWallets()
	if err != nil {
		fmt.Println("Failed to load wallets:", err)
		return
	}
	for _, addr := range ws.GetAddresses() {
		fmt.Println(addr)
	}
}

func (c *CLI) Run() {
	c.validateArgs()

	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)

	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to receive genesis reward (not used yet)")
	getBalanceAddress := getBalanceCmd.String("address", "", "The address")
	sendFrom := sendCmd.String("from", "", "Source address")
	sendTo := sendCmd.String("to", "", "Destination address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")
	startNodeMiner := startNodeCmd.String("miner", "", "Miner address (optional)")

	switch os.Args[1] {
	case "createwallet":
		_ = createWalletCmd.Parse(os.Args[2:])
	case "listaddresses":
		_ = listAddressesCmd.Parse(os.Args[2:])
	case "createblockchain":
		_ = createBlockchainCmd.Parse(os.Args[2:])
	case "printchain":
		_ = printChainCmd.Parse(os.Args[2:])
	case "getbalance":
		_ = getBalanceCmd.Parse(os.Args[2:])
	case "send":
		_ = sendCmd.Parse(os.Args[2:])
	case "startnode":
		_ = startNodeCmd.Parse(os.Args[2:])
	default:
		c.printUsage()
		os.Exit(1)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			fmt.Println("Error: -address is required")
			createBlockchainCmd.Usage()
			os.Exit(1)
		}
		c.createBlockchain(*createBlockchainAddress)
	}

	if createWalletCmd.Parsed() {
		c.createWallet()
	}

	if listAddressesCmd.Parsed() {
		c.listAddresses()
	}

	if printChainCmd.Parsed() {
		c.printChain()
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			fmt.Println("Error: -address is required")
			getBalanceCmd.Usage()
			os.Exit(1)
		}
		c.getBalance(*getBalanceAddress)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			fmt.Println("Error: -from, -to, and -amount (>0) are required")
			sendCmd.Usage()
			os.Exit(1)
		}
		c.send(*sendFrom, *sendTo, *sendAmount)
	}

	if startNodeCmd.Parsed() {
		c.startNode(*startNodeMiner)
	}
}
