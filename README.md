# my-blockchain (Go)

A simplified “mini-Bitcoin” built from scratch in Go

It includes:
- Blocks with PoW header fields (`PrevHash`, `MerkleRoot`, `Timestamp`, `Nonce`, `Hash`)
- Transactions (UTXO-style) with ECDSA (P-256) signatures
- Merkle tree root over transactions
- Persistence using BoltDB (`go.etcd.io/bbolt`)
- A CLI for common actions
- Simple P2P networking (3 local nodes) to sync blocks

> Note: This is a project, not a production blockchain.

## Requirements

- Go 1.22+ (your repo currently uses `go 1.23` in `go.mod`)
- Windows PowerShell (commands below), or adapt to Bash

## Project layout

- `core/` — block, chain, PoW, transactions, UTXO, merkle
- `wallet/` — keypairs + Base58Check addresses, `wallets.dat`
- `network/` — TCP P2P sync
- `cli/` — command line interface

## Data files

- Per-node DB files: `blockchain_<NODE_ID>.db` (example: `blockchain_3000.db`)
- Wallet file (shared by all nodes in the same folder): `wallets.dat`

## Important note (Windows / BoltDB locking)

On Windows, BoltDB uses file locks that can block other processes from opening the same DB file while `startnode` is running.

To make the demo smooth (and closer to Bitcoin’s `bitcoind` + `bitcoin-cli` model), this project adapted **RPC-style requests**:
- `send` asks the running node to build/sign/mine the transaction.
- `getbalance` and `printchain` ask the running node to read state.

If no node is running, the CLI falls back to direct DB access for single-process/offline usage.

## CLI Commands

All commands are run from the `my-blockchain` folder:

```powershell
Set-Location "C:\PATH\my-blockchain"
```

### Wallet

Create a new wallet address:

```powershell
go run . createwallet
```

List saved addresses:

```powershell
go run . listaddresses
```

### Create blockchain (genesis)

Create a fresh chain for the current node (requires `NODE_ID` and an address to receive the genesis coinbase):

```powershell
$env:NODE_ID = "3000"
Remove-Item -ErrorAction SilentlyContinue blockchain_3000.db

go run . createblockchain -address YOUR_ADDRESS
```

### Print chain

```powershell
$env:NODE_ID = "3000"
go run . printchain
```

### Get balance

```powershell
$env:NODE_ID = "3000"
go run . getbalance -address YOUR_ADDRESS
```

### Send transaction (and mine)

If a node is running for the current `NODE_ID`, `send` submits a request to that node, and the **node mines a new block**.

If no node is running, the CLI falls back to local mining (single-process/offline mode).

```powershell
$env:NODE_ID = "3000"
go run . send -from FROM_ADDRESS -to TO_ADDRESS -amount 5
```

## Multi-node (3 terminals) demo

This simulates 3 nodes on one machine listening on ports `3000`, `3001`, `3002`.

### 1) Terminal A — Node 3000 (bootstrap)

```powershell
Set-Location "C:\PATH\my-blockchain"
$env:NODE_ID = "3000"

# (Optional) create addresses
# go run . createwallet
# go run . listaddresses

Remove-Item -ErrorAction SilentlyContinue blockchain_3000.db

go run . createblockchain -address YOUR_ADDRESS

go run . startnode -miner YOUR_ADDRESS
```

### 2) Terminal B — Node 3001

```powershell
Set-Location "C:\PATH\my-blockchain"
$env:NODE_ID = "3001"

Remove-Item -ErrorAction SilentlyContinue blockchain_3001.db

go run . startnode
```

### 3) Terminal C — Node 3002

```powershell
Set-Location "C:\PATH\my-blockchain"
$env:NODE_ID = "3002"

Remove-Item -ErrorAction SilentlyContinue blockchain_3002.db

go run . startnode
```

### 4) Mine a block on node 3000 and watch others sync

In a 4th terminal (recommended, so you don’t stop the node):

```powershell
Set-Location "C:\PATH\my-blockchain"
$env:NODE_ID = "3000"

go run . send -from FROM_ADDRESS -to TO_ADDRESS -amount 1
```

Then verify node 3001 / 3002 synced:

```powershell
$env:NODE_ID = "3001"
go run . printchain

$env:NODE_ID = "3002"
go run . printchain
```
