// Copyright (c) 2026 AldianOkto. All rights reserved.
// Copyright (c) 2026 Eterbit Core.
// Use of this source code is governed by the Apache License.
// that can be found in the root directory of this repository.
// Project: Eterbit / Blockchain Core
//
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at. <http://www.apache.org/licenses/LICENSE-2.0>
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"eterbit/core"
	"eterbit/internal"
	"eterbit/internal/p2p"
	"eterbit/node"
	"eterbit/storage/wallet"

	_ "github.com/cloudflare/circl/sign/dilithium/mode3"
)

// MempoolFile designates the absolute or relative file path utilized for persisting unconfirmed network transaction queues locally.
const MempoolFile = "eterbit_data/mempool.json"

// main serves as the primary entry point for the command-line interface application, parsing operational arguments and routing execution flows accordingly.
func main() {
	// Initialize distinct command-line flag sets for various administrative and operational subcommands.
	walletCreateCmd := flag.NewFlagSet("create", flag.ExitOnError)
	balanceCmd := flag.NewFlagSet("balance", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	nodeCmd := flag.NewFlagSet("node", flag.ExitOnError)
	explorerCmd := flag.NewFlagSet("explorer", flag.ExitOnError)
	mineCmd := flag.NewFlagSet("mine", flag.ExitOnError)
	peersCmd := flag.NewFlagSet("peers", flag.ExitOnError)
	feesCmd := flag.NewFlagSet("fees", flag.ExitOnError)
	getBlockHashCmd := flag.NewFlagSet("getblockhash", flag.ExitOnError)
	getBlockCmd := flag.NewFlagSet("getblock", flag.ExitOnError)
	uptimeCmd := flag.NewFlagSet("uptime", flag.ExitOnError)

	// Define specific parameter bindings for individual command flags.
	walletName := walletCreateCmd.String("name", "keystore.json", "Custom filename for the wallet")

	sendRecipient := sendCmd.String("to", "", "Recipient destination address")
	sendAmount := sendCmd.Uint64("amount", 0, "Transfer value amount")
	sendFee := sendCmd.Uint64("fee", 2, "Transaction fee")
	walletSource := sendCmd.String("wallet", "keystore.json", "Wallet filename to use")

	nodePort := nodeCmd.String("port", ":8333", "P2P listening port for the node")
	nodeConnect := nodeCmd.String("connect", "", "Peer address to connect (e.g., localhost:8333)")

	// Validate whether adequate command-line arguments have been provided by the executing operator.
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Switch execution branch based upon the primary subcommand supplied in system arguments.
	switch os.Args[1] {
	case "create":
		walletCreateCmd.Parse(os.Args[2:])
		handleCreateWallet(*walletName)
	case "balance":
		balanceCmd.Parse(os.Args[2:])
		handleCheckBalance()
	case "send":
		sendCmd.Parse(os.Args[2:])
		handleSendTx(*sendRecipient, *sendAmount, *sendFee, *walletSource)
	case "node":
		nodeCmd.Parse(os.Args[2:])
		handleRunNode(*nodePort, *nodeConnect)
	case "explorer":
		explorerCmd.Parse(os.Args[2:])
		handleExploreBlockchain()
	case "mine":
		mineCmd.Parse(os.Args[2:])
		handleManualMine()
	case "peers":
		peersCmd.Parse(os.Args[2:])
		handleCheckPeers()
	case "fees":
		feesCmd.Parse(os.Args[2:])
		handleCheckFees()
	case "uptime":
		uptimeCmd.Parse(os.Args[2:])
		handleCheckUptime()
	case "getblockhash":
		// Verify that the required block index parameter has been adequately supplied prior to command parsing.
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run eterbit.go getblockhash <block_index>")
			os.Exit(1)
		}
		getBlockHashCmd.Parse(os.Args[2:])
		handleGetBlockHash(os.Args[2])
	case "getblock":
		// Verify that the required block hash parameter has been adequately supplied prior to command parsing.
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run eterbit.go getblock <block_hash>")
			os.Exit(1)
		}
		getBlockCmd.Parse(os.Args[2:])
		handleGetBlock(os.Args[2])
	default:
		printUsage()
		os.Exit(1)
	}
}

// printUsage outputs the standard command-line manual instructions and available command options to standard output.
func printUsage() {
	// Render comprehensive structural documentation regarding available console commands.
	fmt.Println("================================================================================")
	fmt.Println(" ETERBIT BLOCKCHAIN CLI MANAGER (BITCOIN-LIKE ARCHITECTURE)")
	fmt.Println("================================================================================")
	fmt.Println("Available commands:")
	fmt.Println("  go run eterbit.go create -name <file.json>")
	fmt.Println("  go run eterbit.go balance")
	fmt.Println("  go run eterbit.go send -to <addr> -amount <val> [-fee <val>] [-wallet <file>]")
	fmt.Println("  go run eterbit.go node [--port :port] [--connect host:port]")
	fmt.Println("  go run eterbit.go mine")
	fmt.Println("  go run eterbit.go explorer")
	fmt.Println("  go run eterbit.go peers")
	fmt.Println("  go run eterbit.go fees")
	fmt.Println("  go run eterbit.go uptime")
	fmt.Println("  go run eterbit.go getblockhash <index>")
	fmt.Println("  go run eterbit.go getblock <hash>")
	fmt.Println("================================================================================")
}

// handleCreateWallet generates a new cryptographic wallet instance and persists its state securely to disk.
func handleCreateWallet(filename string) {
	// Ensure that the target directory structure exists on disk prior to file operations.
	os.MkdirAll("eterbit_data", 0755)
	filePath := filepath.Join("eterbit_data", filename)

	// Check if a wallet file already exists to prevent accidental overwriting of existing keys.
	if _, err := os.Stat(filePath); err == nil {
		fmt.Printf("[WALLET] Wallet file '%s' already exists!\n", filename)
		return
	}

	// Invoke custom wallet creation logic to generate cryptographic keys and address strings.
	addr, privKey, pubBytes, err := wallet.CreateOrLoadWalletCustom(filePath)
	if err != nil {
		fmt.Printf("[WALLET] Failed: %v\n", err)
		return
	}

	// Encode public and private keys into hexadecimal format for storage serialization.
	pubHex := hex.EncodeToString(pubBytes)
	privBytes, _ := privKey.MarshalBinary()
	privHex := hex.EncodeToString(privBytes)

	// Persist the newly generated wallet parameters securely to the specified file path.
	wallet.SaveWalletCustom(filePath, addr, pubHex, privHex)

	// Output successful wallet generation details to standard output for operator confirmation.
	fmt.Println("================================================================================")
	fmt.Println(" ETERBIT NEW WALLET CREATED")
	fmt.Println("================================================================================")
	fmt.Printf(" Filename : %s\n", filePath)
	fmt.Printf(" Address  : %s\n", addr)
	fmt.Println("--------------------------------------------------------------------------------")
}

// handleCheckBalance queries the persistent ledger database to enumerate all registered account balances and associated nonces.
func handleCheckBalance() {
	// Initialize the ledger state instance from storage utilizing default configuration parameters.
	ledger := node.InitializeLedger("eterbit_data", 3, "SYSTEM_VIEWER")
	fmt.Println("================================================================================")
	fmt.Println(" REGISTERED ACCOUNT BALANCES IN LEVELDB")
	fmt.Println("================================================================================")
	
	// Validate whether any account states currently exist within the ledger database.
	if len(ledger.State) == 0 {
		fmt.Println(" No accounts currently recorded in the state ledger.")
		return
	}
	
	// Iterate through all recorded accounts and print their corresponding balances and nonces.
	for addr, acc := range ledger.State {
		fmt.Printf(" Address: %s | Balance: %.8f Coins | Nonce: %d\n", addr, node.ToDecimal(acc.Balance), acc.Nonce)
	}
	fmt.Println("================================================================================")
}

// saveMempoolToDisk serializes the active transaction pool collection and writes the resulting data structure directly to storage.
func saveMempoolToDisk(mempool []*core.Transfer) {
	// Ensure that the target directory exists before attempting to write mempool data.
	os.MkdirAll("eterbit_data", 0755)
	// Marshal the mempool transaction slice into indented JSON format.
	data, _ := json.MarshalIndent(mempool, "", "  ")
	// Write the serialized byte stream directly to the designated mempool file path.
	os.WriteFile(MempoolFile, data, 0644)
}

// loadMempoolFromDisk reads the serialized transaction pool dataset from disk and unmarshals it into memory.
func loadMempoolFromDisk() []*core.Transfer {
	var mempool []*core.Transfer
	// Read the raw byte content from the persistent mempool storage file.
	data, err := os.ReadFile(MempoolFile)
	if err != nil {
		return mempool
	}
	// Unmarshal the retrieved JSON data back into a slice of transfer pointers.
	json.Unmarshal(data, &mempool)
	return mempool
}

// handleSendTx constructs, signs, and broadcasts a new value transfer transaction into the network mempool architecture.
func handleSendTx(recipient string, amount uint64, fee uint64, walletFile string) {
	// Validate that essential transfer parameters have been provided by the caller.
	if recipient == "" || amount == 0 {
		fmt.Println("[CLI] Incomplete arguments! Use -to and -amount.")
		return
	}

	// Fallback to default keystore filename if no specific wallet file was provided.
	if walletFile == "" {
		walletFile = "keystore.json"
	}

	filePath := filepath.Join("eterbit_data", walletFile)
	addrMiner, _, _, _ := wallet.LoadWalletCustom(filePath)
	
	// Initialize the blockchain ledger context using the derived wallet address.
	ledger := node.InitializeLedger("eterbit_data", 3, addrMiner)
	
	// Load the signing wallet credentials from disk for transaction authentication.
	addrA, privKeyA, pubBytesA, err := wallet.LoadWalletCustom(filePath)
	if err != nil {
		fmt.Printf("[CLI] Failed to load wallet from %s: %v\n", filePath, err)
		return
	}

	// Populate initial state balances for the sender account to ensure valid transaction validation.
	ledger.State[addrA] = &node.AccountState{
		Balance: node.InitialAirdrop,
		Nonce:   0,
	}
	
	// Handle alternative address prefix variations for ledger state compatibility.
	if len(addrA) > 4 && addrA[:4] == "etrb" {
		ledger.State[addrA[4:]] = &node.AccountState{
			Balance: node.InitialAirdrop,
			Nonce:   0,
		}
	} else {
		ledger.State["etrb"+addrA] = &node.AccountState{
			Balance: node.InitialAirdrop,
			Nonce:   0,
		}
	}

	// Compute the current transaction nonce utilizing ledger state and temporal entropy.
	currentNonce := ledger.State[addrA].Nonce + uint64(time.Now().UnixNano()%100000)

	// Output structural transaction construction details to the console interface.
	fmt.Printf("[CLI] Constructing transaction from %s (via %s) to %s (Amount: %.8f, Fee: %.8f)...\n", addrA, walletFile, recipient, node.ToDecimal(amount), node.ToDecimal(fee))

	// Instantiate and cryptographically sign the new transfer transaction structure.
	tx := core.NewTransfer(privKeyA, pubBytesA, recipient, amount, fee, currentNonce)
	
	// Append the newly created transaction into the local mempool storage file.
	existingMempool := loadMempoolFromDisk()
	existingMempool = append(existingMempool, tx)
	saveMempoolToDisk(existingMempool)

	// Confirm successful broadcast of the transaction into the local mempool subsystem.
	fmt.Printf("[MEMPOOL] Transaction broadcasted to network pool! ID: %s...\n", tx.ComputeID()[:16])
	fmt.Println("[CLI] Transaction waiting for node validator to mine into a block.")
}

// handleRunNode initiates a continuous background validation daemon process that periodically polls and processes pending transactions.
func handleRunNode(port string, connectPeer string) {
	fmt.Println("[SYS] Booting Eterbit Live Node (Bitcoin Core Style)...")
	
	// Record node initialization timestamp for daemon uptime tracking functionality.
	internal.RecordStartTime()
	
	// Load default validator wallet credentials required for block reward distribution.
	addrMiner, _, _, err := wallet.LoadWalletCustom("eterbit_data/keystore.json")
	if err != nil {
		fmt.Printf("[NODE] Failed to load default miner wallet: %v\n", err)
		return
	}

	ledger := node.InitializeLedger("eterbit_data", 3, addrMiner)
	server := p2p.NewServer(port)

	// Spawn a background worker routine to periodically dump connected peer information to disk.
	go func() {
		for {
			time.Sleep(2 * time.Second)
			peerList := server.GetPeerList()
			data, _ := json.MarshalIndent(peerList, "", "  ")
			os.MkdirAll("eterbit_data", 0755)
			os.WriteFile("eterbit_data/peers.json", data, 0644)
		}
	}()

	// Define the network transaction reception callback handler for incoming P2P messages.
	onTx := func(tx *core.Transfer) {
		fmt.Println("[P2P] Received transaction from network peer, adding to mempool...")
		ledger.Mu.Lock()
		ledger.Mempool = append(ledger.Mempool, tx)
		ledger.Mu.Unlock()
		
		diskMempool := loadMempoolFromDisk()
		diskMempool = append(diskMempool, tx)
		saveMempoolToDisk(diskMempool)
	}

	// Define the network block reception callback handler for incoming P2P messages.
	onBlock := func(block *core.LedgerBlock) {
		fmt.Printf("[P2P] Received new block #%d from network peer!\n", block.Index)
	}

	// Start the P2P networking listener server asynchronously in the background.
	go func() {
		if err := server.StartListening(onBlock, onTx); err != nil {
			fmt.Printf("[P2P] Server error: %v\n", err)
		}
	}()

	// Establish an outgoing connection to a specified peer node if the connection flag was provided.
	if connectPeer != "" {
		if err := server.ConnectToPeer(connectPeer); err != nil {
			fmt.Printf("[P2P] Failed to connect to peer %s: %v\n", connectPeer, err)
		}
	}

	// Launch a continuous mining loop daemon to process pending transactions from the mempool.
	go func() {
		for {
			time.Sleep(3 * time.Second)
			diskMempool := loadMempoolFromDisk()
			if len(diskMempool) > 0 {
				ledger.Mu.Lock()
				ledger.Mempool = diskMempool
				ledger.Mu.Unlock()

				fmt.Println("[NODE] Pending transactions detected in mempool. Starting Proof-of-Work...")
				ledger.MineBlock()
				saveMempoolToDisk([]*core.Transfer{})
			}
		}
	}()

	// Output operational node status parameters to standard output.
	fmt.Printf("[NODE] Active validator miner: %s\n", addrMiner)
	fmt.Printf("[NODE] P2P Server listening on %s\n", port)
	fmt.Println("[NODE] Node operational and listening. Press Ctrl+C to terminate.")
	
	// Block execution indefinitely to maintain the live node daemon process.
	select {}
}

// handleManualMine executes a single manual iteration of the Proof-of-Work block mining procedure using accumulated mempool records.
func handleManualMine() {
	fmt.Println("[CLI] Triggering Manual Block Mining...")
	
	// Attempt to load default miner wallet credentials, utilizing a fallback identifier if unavailable.
	addrMiner, _, _, err := wallet.LoadWalletCustom("eterbit_data/keystore.json")
	if err != nil {
		addrMiner = "SYSTEM_MINER"
	}

	ledger := node.InitializeLedger("eterbit_data", 3, addrMiner)
	
	// Synchronize any pending mempool transactions stored on disk into the active ledger mempool queue.
	diskMempool := loadMempoolFromDisk()
	if len(diskMempool) > 0 {
		ledger.Mu.Lock()
		ledger.Mempool = diskMempool
		ledger.Mu.Unlock()
	}

	// Execute the core Proof-of-Work block mining algorithm and clear the processed mempool.
	ledger.MineBlock()
	saveMempoolToDisk([]*core.Transfer{})
}

// handleExploreBlockchain parses and displays structural blockchain blocks and metadata directly from physical storage.
func handleExploreBlockchain() {
	// Delegate blockchain inspection tasks to the core blockchain module implementation.
	core.InspectBlockchain("eterbit_data")
}

// handleCheckPeers displays active connected peers list (Bitcoin-like getpeerinfo).
func handleCheckPeers() {
	fmt.Println("================================================================================")
	fmt.Println(" ETERBIT P2P NETWORK - PEER INFO (GETPEERINFO)")
	fmt.Println("================================================================================")
	
	filePath := filepath.Join("eterbit_data", "peers.json")
	// Read the serialized peer list dataset generated by the active node background routine.
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println(" No active node server running or no peers connected.")
		fmt.Println("================================================================================")
		return
	}

	var peers []string
	// Unmarshal the retrieved peer address dataset into string slices.
	if err := json.Unmarshal(data, &peers); err != nil || len(peers) == 0 {
		fmt.Println(" Connected Peers: 0")
		fmt.Println("================================================================================")
		return
	}

	// Output total connection statistics and itemized peer node addresses.
	fmt.Printf(" Total Connected Peers: %d\n", len(peers))
	fmt.Println("--------------------------------------------------------------------------------")
	for i, peer := range peers {
		fmt.Printf(" [%d] Peer Address: %s (Status: ACTIVE/CONNECTED)\n", i+1, peer)
	}
	fmt.Println("================================================================================")
}

// handleCheckFees displays fee market statistics derived from the active mempool dataset.
func handleCheckFees() {
	addrMiner, _, _, err := wallet.LoadWalletCustom("eterbit_data/keystore.json")
	if err != nil {
		addrMiner = "SYSTEM_VIEWER"
	}

	ledger := node.InitializeLedger("eterbit_data", 3, addrMiner)
	
	// Synchronize disk-based mempool records into memory for accurate fee market analysis.
	diskMempool := loadMempoolFromDisk()
	if len(diskMempool) > 0 {
		ledger.Mu.Lock()
		ledger.Mempool = diskMempool
		ledger.Mu.Unlock()
	}

	// Retrieve statistical fee market indicators from the ledger mempool manager.
	count, highest, avg := ledger.GetMempoolFeeStats()

	// Render comprehensive fee metrics to the command-line interface.
	fmt.Println("================================================================================")
	fmt.Println("                    ETERBIT MEMPOOL FEE MARKET                  ")
	fmt.Println("================================================================================")
	fmt.Printf(" Pending Transactions in Mempool : %d\n", count)
	fmt.Printf(" Highest Priority Fee          : %.8f Coins\n", node.ToDecimal(highest))
	fmt.Printf(" Average Fee                   : %.8f Coins\n", node.ToDecimal(uint64(avg)))
	fmt.Println("================================================================================")
}

// handleCheckUptime computes and displays the active operational duration of the node instance utilizing internal modules.
func handleCheckUptime() {
	// Query the internal time tracking module to obtain formatted system uptime strings.
	_, uptimeFormatted := internal.GetUptime()

	// Render the calculated node uptime details to the console operator.
	fmt.Println("================================================================")
	fmt.Println("                  ETERBIT NODE UPTIME INFO                      ")
	fmt.Println("================================================================")
	fmt.Printf(" Uptime: %s\n", uptimeFormatted)
	fmt.Println("================================================================")
}

// handleGetBlockHash retrieves and outputs the hexadecimal block hash corresponding to the specified numerical index.
func handleGetBlockHash(indexStr string) {
	var index uint64
	// Parse the string-based index parameter into an unsigned 64-bit integer.
	_, err := fmt.Sscanf(indexStr, "%d", &index)
	if err != nil {
		fmt.Printf("[CLI] Invalid block index: %s\n", indexStr)
		return
	}

	ledger := node.InitializeLedger("eterbit_data", 3, "SYSTEM_VIEWER")
	// Verify that the requested block index falls within the valid bounds of the local chain.
	if int(index) >= len(ledger.Chain) {
		fmt.Printf("[CLI] Block index #%d out of range (Total blocks: %d)\n", index, len(ledger.Chain))
		return
	}

	// Extract the target block from the chain and encode its hash into hexadecimal text format.
	block := ledger.Chain[index]
	hashHex := hex.EncodeToString(block.Hash)
	fmt.Println(hashHex)
}

// handleGetBlock retrieves and renders the complete structural block data in JSON format based on the provided target hash.
func handleGetBlock(targetHash string) {
	ledger := node.InitializeLedger("eterbit_data", 3, "SYSTEM_VIEWER")
	
	var foundBlock *core.LedgerBlock = nil
	// Search through the local blockchain chain to locate the block matching the target hash.
	for _, block := range ledger.Chain {
		if hex.EncodeToString(block.Hash) == targetHash {
			foundBlock = block
			break
		}
	}

	// Handle cases where no matching block hash could be located within local storage.
	if foundBlock == nil {
		fmt.Printf("[CLI] Block with hash '%s' not found!\n", targetHash)
		return
	}

	// Marshal the structural block data object into formatted JSON indentation.
	jsonData, err := json.MarshalIndent(foundBlock, "", "  ")
	if err != nil {
		fmt.Printf("[CLI] Failed to format block JSON: %v\n", err)
		return
	}

	// Output the complete structural block JSON payload to the console interface.
	fmt.Println("================================================================")
	fmt.Println("                 ETERBIT BLOCK JSON DATA                        ")
	fmt.Println("================================================================")
	fmt.Println(string(jsonData))
	fmt.Println("================================================================")
}
