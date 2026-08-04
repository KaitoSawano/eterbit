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

// getDataDir returns the external global data directory (~/.eterbit) so that blockchain data 
// and wallet.dat remain completely safe even if the project source folder is deleted or updated.
func getDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "eterbit_data" // Fallback to local if home dir is inaccessible
	}
	dir := filepath.Join(homeDir, ".eterbit")
	os.MkdirAll(dir, 0755)
	return dir
}

// MempoolFile designates the absolute file path utilized for persisting unconfirmed network transaction queues externally.
var MempoolFile = filepath.Join(getDataDir(), "mempool.json")

// main serves as the primary entry point for the command-line interface application, parsing operational arguments and routing execution flows accordingly.
func main() {
	// Initialize distinct command-line flag sets for various administrative and operational subcommands.
	walletCreateCmd := flag.NewFlagSet("create", flag.ExitOnError)
	balanceCmd := flag.NewFlagSet("balance", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	nodeCmd := flag.NewFlagSet("node", flag.ExitOnError)
	explorerCmd := flag.NewFlagSet("explorer", flag.ExitOnError)
	mineCmd := flag.NewFlagSet("mine", flag.ExitOnError)
	miningCmd := flag.NewFlagSet("mining", flag.ExitOnError)
	peersCmd := flag.NewFlagSet("peers", flag.ExitOnError)
	feesCmd := flag.NewFlagSet("fees", flag.ExitOnError)
	getBlockHashCmd := flag.NewFlagSet("getblockhash", flag.ExitOnError)
	getBlockCmd := flag.NewFlagSet("getblock", flag.ExitOnError)
	uptimeCmd := flag.NewFlagSet("uptime", flag.ExitOnError)
	getNetTotalsCmd := flag.NewFlagSet("getnettotals", flag.ExitOnError)

	// Define specific parameter bindings for individual command flags.
	walletLabel := walletCreateCmd.String("label", "Default Account", "Label description for the new multi-wallet account")

	sendRecipient := sendCmd.String("to", "", "Recipient destination address")
	sendAmount := sendCmd.Uint64("amount", 0, "Transfer value amount")
	sendFee := sendCmd.Uint64("fee", 2, "Transaction fee")
	sendSenderAddr := sendCmd.String("from", "", "Specific sender account address within wallet.dat")

	nodePort := nodeCmd.String("port", ":19333", "P2P listening port for the node")
	nodeConnect := nodeCmd.String("connect", "", "Peer address to connect (e.g., localhost:19333)")

	mineBlocks := mineCmd.Int("blocks", 1, "Number of blocks to generate")
	mineAddress := mineCmd.String("address", "", "Target destination address for block reward (Bitcoin-like generatetoaddress)")

	// Validate whether adequate command-line arguments have been provided by the executing operator.
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Switch execution branch based upon the primary subcommand supplied in system arguments.
	switch os.Args[1] {
	case "create":
		walletCreateCmd.Parse(os.Args[2:])
		handleCreateWalletAccount(*walletLabel)
	case "balance":
		balanceCmd.Parse(os.Args[2:])
		handleCheckBalance()
	case "send":
		sendCmd.Parse(os.Args[2:])
		handleSendTx(*sendRecipient, *sendAmount, *sendFee, *sendSenderAddr)
	case "node":
		nodeCmd.Parse(os.Args[2:])
		handleRunNode(*nodePort, *nodeConnect)
	case "explorer":
		explorerCmd.Parse(os.Args[2:])
		handleExploreBlockchain()
	case "mine":
		mineCmd.Parse(os.Args[2:])
		handleManualMine(*mineBlocks, *mineAddress)
	case "mining":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run eterbit.go mining <target_address>")
			os.Exit(1)
		}
		miningCmd.Parse(os.Args[3:])
		handleManualMine(1, os.Args[2])
	case "peers":
		peersCmd.Parse(os.Args[2:])
		handleCheckPeers()
	case "fees":
		feesCmd.Parse(os.Args[2:])
		handleCheckFees()
	case "uptime":
		uptimeCmd.Parse(os.Args[2:])
		handleCheckUptime()
	case "getnettotals":
		getNetTotalsCmd.Parse(os.Args[2:])
		handleGetNetTotals()
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
	fmt.Println(" ETERBIT BLOCKCHAIN CLI MANAGER (BITCOIN-LIKE MULTI-WALLET ARCHITECTURE)")
	fmt.Println("================================================================================")
	fmt.Println("Available commands:")
	fmt.Println("  go run eterbit.go create [-label <account_label>]")
	fmt.Println("  go run eterbit.go balance")
	fmt.Println("  go run eterbit.go send -to <addr> -amount <val> [-fee <val>] [-from <sender_addr>]")
	fmt.Println("  go run eterbit.go node [--port :port] [--connect host:port]")
	fmt.Println("  go run eterbit.go mine [-blocks <num>] [-address <addr>]")
	fmt.Println("  go run eterbit.go mining <target_address>")
	fmt.Println("  go run eterbit.go explorer")
	fmt.Println("  go run eterbit.go peers")
	fmt.Println("  go run eterbit.go fees")
	fmt.Println("  go run eterbit.go uptime")
	fmt.Println("  go run eterbit.go getnettotals")
	fmt.Println("  go run eterbit.go getblockhash <index>")
	fmt.Println("  go run eterbit.go getblock <hash>")
	fmt.Println("================================================================================")
}

// handleCreateWalletAccount provisions a new account address inside the centralized wallet.dat container.
func handleCreateWalletAccount(label string) {
	dataDir := getDataDir()
	filePath := filepath.Join(dataDir, "wallet.dat")

	// Generate and append a new account keypair inside wallet.dat
	newAddr, err := wallet.GenerateNewAccount(filePath)
	if err != nil {
		fmt.Printf("[WALLET] Failed to generate new account: %v\n", err)
		return
	}

	fmt.Println("================================================================================")
	fmt.Println(" ETERBIT NEW WALLET ACCOUNT CREATED (WALLET.DAT)")
	fmt.Println("================================================================================")
	fmt.Printf(" Filepath : %s\n", filePath)
	fmt.Printf(" Label    : %s\n", label)
	fmt.Printf(" Address  : %s\n", newAddr)
	fmt.Println("--------------------------------------------------------------------------------")
}

// handleCheckBalance queries the persistent ledger database to enumerate all registered account balances and associated nonces.
func handleCheckBalance() {
	// Initialize the ledger state instance from external storage utilizing default configuration parameters.
	ledger := node.InitializeLedger(getDataDir(), 3, "SYSTEM_VIEWER")
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
	// Marshal the mempool transaction slice into indented JSON format.
	data, _ := json.MarshalIndent(mempool, "", "  ")
	// Write the serialized byte stream directly to the designated mempool file path.
	os.WriteFile(filepath.Join(getDataDir(), "mempool.json"), data, 0644)
}

// loadMempoolFromDisk reads the serialized transaction pool dataset from disk and unmarshals it into memory.
func loadMempoolFromDisk() []*core.Transfer {
	var mempool []*core.Transfer
	mempoolFilePath := filepath.Join(getDataDir(), "mempool.json")
	// Read the raw byte content from the persistent mempool storage file.
	data, err := os.ReadFile(mempoolFilePath)
	if err != nil {
		return mempool
	}
	// Unmarshal the retrieved JSON data back into a slice of transfer pointers.
	json.Unmarshal(data, &mempool)
	return mempool
}

// handleSendTx constructs, signs, and broadcasts a new value transfer transaction into the network mempool architecture.
func handleSendTx(recipient string, amount uint64, fee uint64, senderAddr string) {
	// Validate that essential transfer parameters have been provided by the caller.
	if recipient == "" || amount == 0 {
		fmt.Println("[CLI] Incomplete arguments! Use -to and -amount.")
		return
	}

	dataDir := getDataDir()
	filePath := filepath.Join(dataDir, "wallet.dat")
	
	// Load the centralized multi-wallet container (wallet.dat).
	wf, err := wallet.LoadWalletCustom(filePath)
	if err != nil || wf == nil || len(wf.Accounts) == 0 {
		fmt.Printf("[CLI] Failed to load wallet.dat container: %v\n", err)
		return
	}

	// Determine which account inside wallet.dat to use as the sender
	var selectedAccount *wallet.Account
	if senderAddr != "" {
		for _, acc := range wf.Accounts {
			if acc.Address == senderAddr {
				selectedAccount = &acc
				break
			}
		}
		if selectedAccount == nil {
			fmt.Printf("[CLI] Sender address %s not found in wallet.dat!\n", senderAddr)
			return
		}
	} else {
		// Default to the first account in wallet.dat if no sender address is specified
		selectedAccount = &wf.Accounts[0]
	}

	privKeyPtr, pubBytes, err := wallet.GetPrivateKeyInstance(wf, selectedAccount.Address)
	if err != nil {
		fmt.Printf("[CLI] Failed to load private key for address %s: %v\n", selectedAccount.Address, err)
		return
	}

	// Initialize the blockchain ledger context using the derived wallet address.
	ledger := node.InitializeLedger(dataDir, 3, selectedAccount.Address)

	// Populate initial state balances for the sender account to ensure valid transaction validation.
	ledger.State[selectedAccount.Address] = &node.AccountState{
		Balance: node.InitialAirdrop,
		Nonce:   0,
	}
	
	// Handle alternative address prefix variations for ledger state compatibility.
	if len(selectedAccount.Address) > 4 && selectedAccount.Address[:4] == "etrb" {
		ledger.State[selectedAccount.Address[4:]] = &node.AccountState{
			Balance: node.InitialAirdrop,
			Nonce:   0,
		}
	} else {
		ledger.State["etrb"+selectedAccount.Address] = &node.AccountState{
			Balance: node.InitialAirdrop,
			Nonce:   0,
		}
	}

	// Compute the current transaction nonce utilizing ledger state and temporal entropy.
	currentNonce := ledger.State[selectedAccount.Address].Nonce + uint64(time.Now().UnixNano()%100000)

	// Output structural transaction construction details to the console interface.
	fmt.Printf("[CLI] Constructing transaction from %s (wallet.dat) to %s (Amount: %.8f, Fee: %.8f)...\n", selectedAccount.Address, recipient, node.ToDecimal(amount), node.ToDecimal(fee))

	// Instantiate and cryptographically sign the new transfer transaction structure.
	tx := core.NewTransfer(privKeyPtr, pubBytes, recipient, amount, fee, currentNonce)
	
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
	
	// Load default validator miner wallet credentials from wallet.dat for block reward distribution.
	wf, err := wallet.LoadWallet()
	var addrMiner string
	if err != nil || wf == nil || len(wf.Accounts) == 0 {
		addrMiner = "SYSTEM_MINER"
	} else {
		addrMiner = wf.Accounts[0].Address
	}

	dataDir := getDataDir()
	ledger := node.InitializeLedger(dataDir, 3, addrMiner)
	server := p2p.NewServer(port)

	// Register NetTotals HTTP endpoint handler on the P2P server mux
	internal.RegisterNetTotalsHandler(server.Mux())

	// Spawn a background worker routine to periodically dump connected peer information & addrman discovered addresses to disk.
	go func() {
		for {
			time.Sleep(2 * time.Second)
			peerList := server.GetPeerList()
			data, _ := json.MarshalIndent(peerList, "", "  ")
			os.WriteFile(filepath.Join(dataDir, "peers.json"), data, 0644)
			
			// If Server implements AddrManager inspection, persist or log active discovered addresses here as well.
			if server.AddrManager != nil {
				knownAddrs := server.AddrManager.GetKnownAddresses()
				addrData, _ := json.MarshalIndent(knownAddrs, "", "  ")
				os.WriteFile(filepath.Join(dataDir, "addrman_peers.json"), addrData, 0644)
			}
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

	// Automatically discover peers via Hardcoded Seeds and DNS Seeds (BIP 155 style), or fallback to manual override.
	server.AutoDiscoverAndConnect(connectPeer)

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

// handleManualMine executes iterative Proof-of-Work block mining targeting a specific reward address (generatetoaddress style).
func handleManualMine(blocksCount int, targetAddress string) {
	wf, err := wallet.LoadWallet()
	
	if targetAddress == "" {
		if err != nil || wf == nil || len(wf.Accounts) == 0 {
			targetAddress = "SYSTEM_MINER"
		} else {
			targetAddress = wf.Accounts[0].Address
		}
	}

	fmt.Printf("[CLI] Triggering Manual Block Mining (Target Address: %s)...\n", targetAddress)

	dataDir := getDataDir()
	ledger := node.InitializeLedger(dataDir, 3, targetAddress)
	
	// Synchronize any pending mempool transactions stored on disk into the active ledger mempool queue.
	diskMempool := loadMempoolFromDisk()
	if len(diskMempool) > 0 {
		ledger.Mu.Lock()
		ledger.Mempool = diskMempool
		ledger.Mu.Unlock()
	}

	for i := 0; i < blocksCount; i++ {
		fmt.Printf("[NODE] Mining block %d of %d...\n", i+1, blocksCount)
		ledger.MineBlock()
	}
	
	saveMempoolToDisk([]*core.Transfer{})
	fmt.Println("[CLI] Mining completed successfully!")
}

// handleExploreBlockchain parses and displays structural blockchain blocks and metadata directly from physical storage.
func handleExploreBlockchain() {
	// Delegate blockchain inspection tasks to the core blockchain module implementation.
	core.InspectBlockchain(getDataDir())
}

// handleCheckPeers displays active connected peers list (Bitcoin-like getpeerinfo).
func handleCheckPeers() {
	fmt.Println("================================================================================")
	fmt.Println(" ETERBIT P2P NETWORK - PEER INFO (GETPEERINFO)")
	fmt.Println("================================================================================")
	
	filePath := filepath.Join(getDataDir(), "peers.json")
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
	wf, err := wallet.LoadWallet()
	var addrMiner string
	if err != nil || wf == nil || len(wf.Accounts) == 0 {
		addrMiner = "SYSTEM_VIEWER"
	} else {
		addrMiner = wf.Accounts[0].Address
	}

	ledger := node.InitializeLedger(getDataDir(), 3, addrMiner)
	
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
	fmt.Println("                        ETERBIT MEMPOOL FEE MARKET                 ")
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

// handleGetNetTotals retrieves and displays network traffic statistics (Bitcoin-like getnettotals).
func handleGetNetTotals() {
	totals := internal.GetNetTotals()
	out, _ := json.MarshalIndent(totals, "", "  ")
	fmt.Println(string(out))
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

	ledger := node.InitializeLedger(getDataDir(), 3, "SYSTEM_VIEWER")
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
	ledger := node.InitializeLedger(getDataDir(), 3, "SYSTEM_VIEWER")
	
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
	fmt.Println("                  ETERBIT BLOCK JSON DATA                       ")
	fmt.Println("================================================================")
	fmt.Println(string(jsonData))
	fmt.Println("================================================================")
}
