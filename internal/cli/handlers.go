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

package cli

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"eterbit/core"
	"eterbit/internal"
	"eterbit/internal/p2p"
	"eterbit/node"
	"eterbit/storage/wallet"
)

// GetDataDir returns the external global data directory (~/.eterbit) so that blockchain data
// and wallet configurations remain safe even if the source repository folder is modified or removed.
func GetDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "eterbit_data" // Fallback directory if home path is inaccessible.
	}
	dir := filepath.Join(homeDir, ".eterbit")
	os.MkdirAll(dir, 0755)
	return dir
}

// SaveMempoolToDisk serializes the active transaction queue and persists it to external disk storage.
func SaveMempoolToDisk(mempool []*core.Transfer) {
	data, _ := json.MarshalIndent(mempool, "", "  ")
	os.WriteFile(filepath.Join(GetDataDir(), "mempool.json"), data, 0644)
}

// LoadMempoolFromDisk reads the serialized transaction pool dataset from disk and unmarshals it into memory.
func LoadMempoolFromDisk() []*core.Transfer {
	var mempool []*core.Transfer
	mempoolFilePath := filepath.Join(GetDataDir(), "mempool.json")
	data, err := os.ReadFile(mempoolFilePath)
	if err != nil {
		return mempool
	}
	json.Unmarshal(data, &mempool)
	return mempool
}

// HandleCreateWalletAccount provisions a new cryptographic keypair account inside the centralized wallet container.
func HandleCreateWalletAccount(label string) {
	dataDir := GetDataDir()
	filePath := filepath.Join(dataDir, "wallet.dat")
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

// HandleCheckBalance queries the state database to display all registered account balances and nonces.
func HandleCheckBalance() {
	ledger := node.InitializeLedger(GetDataDir(), 3, "SYSTEM_VIEWER")
	fmt.Println("================================================================================")
	fmt.Println(" REGISTERED ACCOUNT BALANCES IN LEVELDB")
	fmt.Println("================================================================================")
	if len(ledger.State) == 0 {
		fmt.Println(" No accounts currently recorded in the state ledger.")
		return
	}
	for addr, acc := range ledger.State {
		fmt.Printf(" Address: %s | Balance: %.8f Coins | Nonce: %d\n", addr, node.ToDecimal(acc.Balance), acc.Nonce)
	}
	fmt.Println("================================================================================")
}

// HandleCheckSupply queries the blockchain ledger and displays maximum supply and circulating minted coins.
func HandleCheckSupply() {
	// Define the fixed maximum coin supply limit updated to 785 million coins (raw uint64 with 8 decimals)
	const maxSupply uint64 = 785000000 * 100000000
	const rewardPerBlock uint64 = 5000000000 // 50 coins per block with 8 decimals precision

	// Initialize the ledger instance to inspect total minted blocks
	ledger := node.InitializeLedger(GetDataDir(), 3, "SYSTEM_VIEWER")
	
	// Calculate circulating supply based on total mined blocks
	totalBlocks := uint64(len(ledger.Chain))
	circulatingSupply := totalBlocks * rewardPerBlock

	// Render the supply metrics report to standard output
	fmt.Println("================================================================================")
	fmt.Println("                     ETERBIT COIN SUPPLY STATISTICS                             ")
	fmt.Println("================================================================================")
	fmt.Printf(" Max Supply         : %.8f Coins\n", node.ToDecimal(maxSupply))
	fmt.Printf(" Circulating Supply : %.8f Coins\n", node.ToDecimal(circulatingSupply))
	if maxSupply > circulatingSupply {
		remaining := maxSupply - circulatingSupply
		fmt.Printf(" Remaining to Mine  : %.8f Coins\n", node.ToDecimal(remaining))
	} else {
		fmt.Println(" Remaining to Mine  : 0.00000000 Coins (Fully Minted)")
	}
	fmt.Println("================================================================================")
}

// HandleSendTx constructs, signs, and broadcasts a new value transfer transaction to the local mempool storage.
func HandleSendTx(recipient string, amount uint64, fee uint64, senderAddr string) {
	if recipient == "" || amount == 0 {
		fmt.Println("[CLI] Incomplete arguments! Use -to and -amount.")
		return
	}
	dataDir := GetDataDir()
	filePath := filepath.Join(dataDir, "wallet.dat")
	wf, err := wallet.LoadWalletCustom(filePath)
	if err != nil || wf == nil || len(wf.Accounts) == 0 {
		fmt.Printf("[CLI] Failed to load wallet.dat container: %v\n", err)
		return
	}

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
		selectedAccount = &wf.Accounts[0]
	}

	privKeyPtr, pubBytes, err := wallet.GetPrivateKeyInstance(wf, selectedAccount.Address)
	if err != nil {
		fmt.Printf("[CLI] Failed to load private key for address %s: %v\n", selectedAccount.Address, err)
		return
	}

	ledger := node.InitializeLedger(dataDir, 3, selectedAccount.Address)
	ledger.State[selectedAccount.Address] = &node.AccountState{Balance: node.InitialAirdrop, Nonce: 0}
	
	currentNonce := ledger.State[selectedAccount.Address].Nonce + uint64(time.Now().UnixNano()%100000)

	fmt.Printf("[CLI] Constructing transaction from %s to %s (Amount: %.8f, Fee: %.8f)...\n", selectedAccount.Address, recipient, node.ToDecimal(amount), node.ToDecimal(fee))

	tx := core.NewTransfer(privKeyPtr, pubBytes, recipient, amount, fee, currentNonce)
	existingMempool := LoadMempoolFromDisk()
	existingMempool = append(existingMempool, tx)
	SaveMempoolToDisk(existingMempool)

	fmt.Printf("[MEMPOOL] Transaction broadcasted! ID: %s...\n", tx.ComputeID()[:16])
}

// HandleAddNode allows manual addition of a peer address to the addrman database.
func HandleAddNode(peerAddr string) {
	if peerAddr == "" {
		if len(os.Args) > 2 {
			peerAddr = os.Args[2]
		} else {
			fmt.Println("[CLI] Error: Target peer address is required. Usage: go run eterbit.go addnode <host:port>")
			return
		}
	}

	dataDir := GetDataDir()
	host, portStr, err := net.SplitHostPort(peerAddr)
	if err != nil {
		fmt.Printf("[CLI] Invalid address format '%s'. Please use host:port (e.g., 127.0.0.1:19333)\n", peerAddr)
		return
	}

	var port int
	fmt.Sscanf(portStr, "%d", &port)

	am := p2p.NewAddrManager(dataDir)
	am.AddAddress(host, port)

	fmt.Println("================================================================================")
	fmt.Println(" ETERBIT P2P NETWORK - ADDNODE MANUAL REGISTRATION")
	fmt.Println("================================================================================")
	fmt.Printf(" Successfully added peer to addrman database: %s:%d\n", host, port)
	fmt.Printf(" Persisted to : %s/peers_addrman.json\n", dataDir)
	fmt.Println("================================================================================")
}

// HandleManualMine executes iterative Proof-of-Work block mining targeting a specific reward address.
func HandleManualMine(blocksCount int, targetAddress string) {
	wf, err := wallet.LoadWallet()
	if targetAddress == "" {
		if err != nil || wf == nil || len(wf.Accounts) == 0 {
			targetAddress = "SYSTEM_MINER"
		} else {
			targetAddress = wf.Accounts[0].Address
		}
	}

	fmt.Printf("[CLI] Triggering Manual Block Mining (Target Address: %s)...\n", targetAddress)
	dataDir := GetDataDir()
	ledger := node.InitializeLedger(dataDir, 3, targetAddress)
	
	diskMempool := LoadMempoolFromDisk()
	if len(diskMempool) > 0 {
		ledger.Mu.Lock()
		ledger.Mempool = diskMempool
		ledger.Mu.Unlock()
	}

	for i := 0; i < blocksCount; i++ {
		fmt.Printf("[NODE] Mining block %d of %d...\n", i+1, blocksCount)
		ledger.MineBlock()
	}
	
	SaveMempoolToDisk([]*core.Transfer{})
	fmt.Println("[CLI] Mining completed successfully!")
}

// HandleExploreBlockchain parses and inspects structural blockchain blocks directly from storage.
func HandleExploreBlockchain() {
	core.InspectBlockchain(GetDataDir())
}

// HandleCheckPeers displays the active connected peers list.
func HandleCheckPeers() {
	fmt.Println("================================================================================")
	fmt.Println(" ETERBIT P2P NETWORK - PEER INFO")
	fmt.Println("================================================================================")
	filePath := filepath.Join(GetDataDir(), "peers.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println(" No active node server running or no peers connected.")
		fmt.Println("================================================================================")
		return
	}

	var peers []string
	if err := json.Unmarshal(data, &peers); err != nil || len(peers) == 0 {
		fmt.Println(" Connected Peers: 0")
		fmt.Println("================================================================================")
		return
	}

	fmt.Printf(" Total Connected Peers: %d\n", len(peers))
	fmt.Println("--------------------------------------------------------------------------------")
	for i, peer := range peers {
		fmt.Printf(" [%d] Peer Address: %s (Status: ACTIVE/CONNECTED)\n", i+1, peer)
	}
	fmt.Println("================================================================================")
}

// HandleCheckFees displays fee market statistics derived from the active mempool dataset.
func HandleCheckFees() {
	wf, err := wallet.LoadWallet()
	addrMiner := "SYSTEM_VIEWER"
	if err == nil && wf != nil && len(wf.Accounts) > 0 {
		addrMiner = wf.Accounts[0].Address
	}

	ledger := node.InitializeLedger(GetDataDir(), 3, addrMiner)
	diskMempool := LoadMempoolFromDisk()
	if len(diskMempool) > 0 {
		ledger.Mu.Lock()
		ledger.Mempool = diskMempool
		ledger.Mu.Unlock()
	}

	count, highest, avg := ledger.GetMempoolFeeStats()
	fmt.Println("================================================================================")
	fmt.Println("                         ETERBIT MEMPOOL FEE MARKET                             ")
	fmt.Println("================================================================================")
	fmt.Printf(" Pending Transactions in Mempool : %d\n", count)
	fmt.Printf(" Highest Priority Fee          : %.8f Coins\n", node.ToDecimal(highest))
	fmt.Printf(" Average Fee                   : %.8f Coins\n", node.ToDecimal(uint64(avg)))
	fmt.Println("================================================================================")
}

// HandleCheckUptime computes and displays the active operational duration of the node instance.
func HandleCheckUptime() {
	_, uptimeFormatted := internal.GetUptime()
	fmt.Println("================================================================")
	fmt.Println("                  ETERBIT NODE UPTIME INFO                      ")
	fmt.Println("================================================================")
	fmt.Printf(" Uptime: %s\n", uptimeFormatted)
	fmt.Println("================================================================")
}

// HandleGetNetTotals retrieves and displays network traffic statistics.
func HandleGetNetTotals() {
	totals := internal.GetNetTotals()
	out, _ := json.MarshalIndent(totals, "", "  ")
	fmt.Println(string(out))
}

// HandleGetBlockHash retrieves and outputs the hexadecimal block hash corresponding to a numerical block index.
func HandleGetBlockHash(indexStr string) {
	var index uint64
	_, err := fmt.Sscanf(indexStr, "%d", &index)
	if err != nil {
		fmt.Printf("[CLI] Invalid block index: %s\n", indexStr)
		return
	}

	ledger := node.InitializeLedger(GetDataDir(), 3, "SYSTEM_VIEWER")
	if int(index) >= len(ledger.Chain) {
		fmt.Printf("[CLI] Block index #%d out of range (Total: %d)\n", index, len(ledger.Chain))
		return
	}

	block := ledger.Chain[index]
	fmt.Println(hex.EncodeToString(block.Hash))
}

// HandleGetBlock retrieves and renders complete structural block data in JSON format based on a target hash.
func HandleGetBlock(targetHash string) {
	ledger := node.InitializeLedger(GetDataDir(), 3, "SYSTEM_VIEWER")
	var foundBlock *core.LedgerBlock = nil
	for _, block := range ledger.Chain {
		if hex.EncodeToString(block.Hash) == targetHash {
			foundBlock = block
			break
		}
	}

	if foundBlock == nil {
		fmt.Printf("[CLI] Block with hash '%s' not found!\n", targetHash)
		return
	}

	jsonData, err := json.MarshalIndent(foundBlock, "", "  ")
	if err != nil {
		return
	}
	fmt.Println("================================================================")
	fmt.Println("                  ETERBIT BLOCK JSON DATA                       ")
	fmt.Println("================================================================")
	fmt.Println(string(jsonData))
	fmt.Println("================================================================")
}
