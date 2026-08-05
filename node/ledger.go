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

package node

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"eterbit/core"
	"eterbit/internal/consensus"
	"eterbit/storage"
)

// Hardcoded Genesis Hash checkpoint linked directly to the immutable consensus specifications.
const HardcodedGenesisHash = "0001a77b043c97b8e43f13c0a2856fe2ecfa1c49ee4105e0fabc9bcbbcb64fcb811ac760dfacdf07bca63c9f87b8b12a524ea83792c7aa9b8041562c02e65677"

// AccountState represents the account balance and transaction sequence nonce.
type AccountState struct {
	Balance uint64 `json:"balance"`
	Nonce   uint64 `json:"nonce"`
}

// LedgerCore manages the blockchain chain state, mempool transaction queue, and block validation engine.
type LedgerCore struct {
	Mu           sync.RWMutex
	Chain        []*core.LedgerBlock
	State        map[string]*AccountState
	Mempool      []*core.Transfer
	Engine       *core.ConsensusEngine
	MinerAddress string
	Storage      *storage.Database
	StopSignal   chan bool
}

// formatCoin converts raw integer units to a floating-point representation for display.
func formatCoin(amount uint64) float64 {
	// Use CoinUnit retrieved directly from the centralized consensus parameters.
	return float64(amount) / float64(consensus.CoinUnit)
}

// CalculateBlockReward delegates to the centralized consensus calculation engine.
func CalculateBlockReward(blockIndex uint64) uint64 {
	return consensus.CalculateBlockReward(blockIndex)
}

// GetTotalCirculatingSupply calculates the total cumulative coins circulating across all existing blocks.
func (lc *LedgerCore) GetTotalCirculatingSupply() uint64 {
	var totalSupply uint64 = 0
	for _, block := range lc.Chain {
		totalSupply += block.Reward
	}
	return totalSupply
}

// VerifyConsensusIntegrity performs a cryptographic and macroeconomic check against the genesis block 
// to ensure local consensus parameters have not been unlawfully altered after storage initialization.
func (lc *LedgerCore) VerifyConsensusIntegrity() {
	if len(lc.Chain) == 0 {
		return
	}
	genesis := lc.Chain[0]

	// Convert raw byte slice hash to hex string format for comparison against the hardcoded checkpoint.
	genesisHashHex := hex.EncodeToString(genesis.Hash)

	// Strict checkpoint validation: Halt immediately if the database hash drifts from the immutable checkpoint.
	if HardcodedGenesisHash != "" && genesisHashHex != HardcodedGenesisHash {
		panic(fmt.Sprintf("\n\n[FATAL CONSENSUS PANIC] GENESIS TAMPERING / RULE MISMATCH DETECTED!\n"+
			"The genesis block or its rules have been modified while an existing database is present!\n"+
			"Expected Checkpoint Hash: %s\n"+
			"Got Database Hash:        %s\n"+
			"Node execution halted immediately to preserve network consensus integrity.\n",
			HardcodedGenesisHash, genesisHashHex))
	}

	// Also leverage the centralized consensus verification function for completeness.
	if err := consensus.VerifyGenesisCheckpoint(genesis.Hash); err != nil {
		panic(fmt.Sprintf("\n[FATAL CONSENSUS PANIC] %v", err))
	}

	// Verify macroeconomic genesis reward invariants against the trusted calculation engine.
	expectedGenesisReward := CalculateBlockReward(0)
	if genesis.Reward != expectedGenesisReward {
		panic(fmt.Sprintf("\n[FATAL CONSENSUS PANIC] DATABASE/CONSENSUS MISMATCH DETECTED!\n"+
			"The immutable macroeconomic rules (MaxSupply/Reward) have been illegally modified!\n"+
			"Stored Genesis Reward: %d | Current Code Genesis Reward: %d\n"+
			"Node execution halted immediately to preserve network consensus integrity.",
			genesis.Reward, expectedGenesisReward))
	}
}

// InitializeLedger initializes or loads the local ledger database state from the specified storage path.
func InitializeLedger(dbPath string, initialDifficulty uint32, minerAddr string) *LedgerCore {
	// Initialize a new persistent storage database instance at the designated directory path.
	db, err := storage.NewDatabase(dbPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to open database: %v", err))
	}

	// Retrieve default consensus parameters for configuring the mining difficulty.
	params := consensus.DefaultConsensus()
	if initialDifficulty == 0 {
		initialDifficulty = uint32(params.DifficultyBits)
	}

	// Instantiate the core ledger management structure with empty state collections and synchronization primitives.
	coreLedger := &LedgerCore{
		Chain:        make([]*core.LedgerBlock, 0),
		State:        make(map[string]*AccountState),
		Mempool:      make([]*core.Transfer, 0),
		Engine:       core.NewConsensusEngine(initialDifficulty),
		MinerAddress: minerAddr,
		Storage:      db,
		StopSignal:   make(chan bool),
	}

	// Attempt to load existing blockchain data from disk; if empty, spawn the network genesis block.
	if !coreLedger.LoadFromDisk() {
		fmt.Println("[DB] Database is empty. Spawning Genesis Block...")
		coreLedger.SpawnGenesis()
	} else {
		fmt.Println("[DB] Blockchain successfully loaded from LevelDB storage!")
	}

	return coreLedger
}

// LoadFromDisk loads existing blockchain blocks from disk storage and rebuilds the account state.
func (lc *LedgerCore) LoadFromDisk() bool {
	// Retrieve the highest committed block height index from the underlying storage database.
	lastIdx, exists := lc.Storage.GetLastIndex()
	if !exists {
		return false
	}

	// Iterate sequentially from the genesis block up to the latest recorded block index.
	for i := uint64(0); i <= lastIdx; i++ {
		data, err := lc.Storage.GetBlock(i)
		if err != nil {
			break
		}
		var block core.LedgerBlock
		// Unmarshal the retrieved block byte payload and append it to the local chain array.
		if err := json.Unmarshal(data, &block); err == nil {
			lc.Chain = append(lc.Chain, &block)
			lc.RebuildState(&block)
		}
	}
	
	// Execute rigid startup integrity checks against unauthorized rule changes.
	lc.VerifyConsensusIntegrity()
	return len(lc.Chain) > 0
}

// RebuildState updates the account balances and nonces based on the transactions within a block.
func (lc *LedgerCore) RebuildState(block *core.LedgerBlock) {
	// Iterate through all transfer transactions recorded within the processed block.
	for _, tx := range block.Transfers {
		sender := hex.EncodeToString(tx.SenderPubKey[:16])
		if _, ok := lc.State[sender]; !ok {
			lc.State[sender] = &AccountState{Balance: 0, Nonce: 0}
		}
		
		// Deduct the transfer value and transaction fee from the sender account balance if sufficient funds exist.
		if lc.State[sender].Balance >= (tx.Value + tx.Fee) {
			lc.State[sender].Balance -= (tx.Value + tx.Fee)
		} else {
			lc.State[sender].Balance = 0
		}
		lc.State[sender].Nonce++

		// Ensure the recipient account state exists within the ledger mapping before crediting funds.
		if _, ok := lc.State[tx.Recipient]; !ok {
			lc.State[tx.Recipient] = &AccountState{Balance: 0, Nonce: 0}
		}
		lc.State[tx.Recipient].Balance += tx.Value
	}

	// Distribute block rewards and accumulated fees to the designated block miner address.
	if block.Miner != "SYSTEM_GENESIS" && block.Miner != "" {
		var feeTotal uint64 = 0
		for _, tx := range block.Transfers {
			feeTotal += tx.Fee
		}
		
		totalRewardAdded := block.Reward + feeTotal
		if totalRewardAdded > 0 {
			if _, ok := lc.State[block.Miner]; !ok {
				lc.State[block.Miner] = &AccountState{Balance: 0, Nonce: 0}
			}
			lc.State[block.Miner].Balance += totalRewardAdded
		}
	}
}

// SpawnGenesis creates and persists the initial genesis block of the blockchain network.
func (lc *LedgerCore) SpawnGenesis() {
	// Compute the base block reward allocation specifically designated for block index zero.
	exactReward := CalculateBlockReward(0)
	
	// --- GENESIS TIMESTAMP MESSAGE ---
	pszTimestamp := "IND Today 05/Aug/2026 Aldianokto, While banks keep printing Debt, We build an honest Exit"

	genesis := &core.LedgerBlock{
		Index:      0,
		Timestamp:  time.Now().Unix(),
		PrevHash:   make([]byte, 64), // 64 bytes to align fully with SHA3-512 standards
		Transfers:  []*core.Transfer{},
		Miner:      "SYSTEM_GENESIS",
		Nonce:      0,
		Difficulty: lc.Engine.TargetDifficulty,
		Reward:     exactReward,
		Message:    pszTimestamp, // Embed genesis message string here
	}
	
	// Execute the consensus mining algorithm to solve the genesis block proof-of-work puzzle.
	_, genesis.Hash = lc.Engine.Mine(genesis)
	genesis.Reward = exactReward // Protect the genesis reward value against external modifications.

	// Append the newly minted genesis block to the local chain array and persist it to storage.
	lc.Chain = append(lc.Chain, genesis)
	lc.Storage.SaveBlock(0, genesis)
	
	fmt.Printf("[GENESIS] Block 0 Created with message: '%s'\n", pszTimestamp)
}

// AddToMempool validates and inserts a transaction payload into the pending mempool queue with Fee Market priority sorting.
func (lc *LedgerCore) AddToMempool(tx *core.Transfer) bool {
	// --- STRICT ADDRESS PREFIX VALIDATION ---
	params := consensus.DefaultConsensus()
	requiredPrefix := params.AddressPrefix // Pure prefix without underscore (e.g., "etrb")
	if !strings.HasPrefix(tx.Recipient, requiredPrefix) {
		fmt.Printf("[MEMPOOL REJECTION] Invalid recipient address prefix: '%s'. Network strictly requires '%s'\n", tx.Recipient, requiredPrefix)
		return false
	}
	// -----------------------------------------------------------------

	lc.Mu.Lock()
	defer lc.Mu.Unlock()

	// Verify the cryptographic signature authenticity of the incoming transfer transaction.
	if !tx.Verify() {
		fmt.Println("[MEMPOOL] Invalid transaction cryptographic signature!")
		return false
	}

	sender := hex.EncodeToString(tx.SenderPubKey[:16])
	acc, exists := lc.State[sender]
	
	// Initialize account states if missing.
	if !exists {
		lc.State[sender] = &AccountState{Balance: 0, Nonce: tx.Nonce}
		acc = lc.State[sender]
	}

	if tx.Nonce != acc.Nonce {
		acc.Nonce = tx.Nonce
	}

	// Insert the validated transaction into the active mempool slice collection.
	lc.Mempool = append(lc.Mempool, tx)

	// Sort the mempool collection to place transactions with higher priority fees at the front for miners.
	sort.Slice(lc.Mempool, func(i, j int) bool {
		return lc.Mempool[i].Fee > lc.Mempool[j].Fee
	})

	fmt.Printf("[MEMPOOL] Transaction successfully queued with Fee: %.8f (ID: %s...)\n", formatCoin(tx.Fee), tx.ComputeID()[:12])
	return true
}

// GetMempoolFeeStats calculates the total, highest, and average fee metrics from transactions residing within the mempool.
func (lc *LedgerCore) GetMempoolFeeStats() (int, uint64, float64) {
	lc.Mu.RLock()
	defer lc.Mu.RUnlock()

	count := len(lc.Mempool)
	if count == 0 {
		return 0, 0, 0
	}

	var totalFee uint64 = 0
	highestFee := lc.Mempool[0].Fee

	// Aggregate total fee values across all pending mempool transaction items.
	for _, tx := range lc.Mempool {
		totalFee += tx.Fee
	}

	avgFee := float64(totalFee) / float64(count)
	return count, highestFee, avgFee
}

// StartLiveWorker starts the background worker daemon to periodically mine blocks from pending mempool transactions or empty blocks.
func (lc *LedgerCore) StartLiveWorker(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				// Trigger block mining operations if pending transactions exist within the mempool queue.
				if len(lc.Mempool) > 0 {
					lc.MineBlock()
				}
			case <-lc.StopSignal:
				ticker.Stop()
				return
			}
		}
	}()
}

// MineBlock packages mempool transactions, executes proof-of-work mining, and appends the new block to the ledger.
func (lc *LedgerCore) MineBlock() {
	// --- STRICT MINER ADDRESS PREFIX VALIDATION ---
	params := consensus.DefaultConsensus()
	if lc.MinerAddress != "SYSTEM_GENESIS" && !strings.HasPrefix(lc.MinerAddress, params.AddressPrefix) {
		panic(fmt.Sprintf("[CONSENSUS VIOLATION] Invalid miner address prefix! Expected prefix '%s', got '%s'", params.AddressPrefix, lc.MinerAddress))
	}
	// ----------------------------------------------

	lc.Mu.Lock()
	parent := lc.Chain[len(lc.Chain)-1]
	validTx := make([]*core.Transfer, 0)
	var feeTotal uint64 = 0

	if len(lc.Mempool) > 0 {
		// Limit the maximum number of transactions processed per block to optimize batch throughput.
		maxTxPerBlock := 10
		limit := len(lc.Mempool)
		if limit > maxTxPerBlock {
			limit = maxTxPerBlock
		}

		// Process each priority transaction, updating sender balances and nonce states accordingly.
		for i := 0; i < limit; i++ {
			tx := lc.Mempool[i]
			sender := hex.EncodeToString(tx.SenderPubKey[:16])
			
			if _, ok := lc.State[sender]; !ok {
				lc.State[sender] = &AccountState{Balance: 0, Nonce: 0}
			}
			acc := lc.State[sender]

			if acc.Balance >= (tx.Value + tx.Fee) {
				acc.Balance -= (tx.Value + tx.Fee)
			} else {
				acc.Balance = 0
			}
			acc.Nonce++

			if _, ok := lc.State[tx.Recipient]; !ok {
				lc.State[tx.Recipient] = &AccountState{Balance: tx.Value, Nonce: 0}
			} else {
				lc.State[tx.Recipient].Balance += tx.Value
			}

			feeTotal += tx.Fee
			validTx = append(validTx, tx)
			fmt.Printf("[MINER] -> Processing Priority Tx: %.8f Coins to %s (Fee: %.8f)\n", formatCoin(tx.Value), tx.Recipient, formatCoin(tx.Fee))
		}
		
		// Trim the mempool queue, retaining unincluded transactions for subsequent block cycles.
		lc.Mempool = lc.Mempool[limit:]
	}
	lc.Mu.Unlock()

	nextIndex := parent.Index + 1
	
	// --- IMMUTABLE MAX SUPPLY VALIDATION ENFORCEMENT ---
	currentSupply := lc.GetTotalCirculatingSupply()
	exactReward := CalculateBlockReward(nextIndex)

	if currentSupply >= consensus.MaxSupply {
		exactReward = 0 // Circulating supply has reached the hard cap limit. Zero reward!
		fmt.Println("[WARNING] MaxSupply hard cap reached! Block reward is now 0.")
	} else if currentSupply+exactReward > consensus.MaxSupply {
		exactReward = consensus.MaxSupply - currentSupply // Trim reward to fit remaining supply quota exactly
	}
	// ----------------------------------------------------

	currentTime := time.Now().Unix()

	// Implement dynamic difficulty adjustment based on elapsed time from parent block.
	calculatedDiff := consensus.CalculateNextDifficulty(parent.Timestamp, currentTime, uint64(parent.Difficulty))
	lc.Engine.TargetDifficulty = uint32(calculatedDiff)

	// Construct the new ledger block container structure with current parameters.
	newBlock := &core.LedgerBlock{
		Index:      nextIndex,
		Timestamp:  currentTime,
		PrevHash:   parent.Hash,
		Transfers:  validTx,
		Miner:      lc.MinerAddress,
		Difficulty: lc.Engine.TargetDifficulty,
		Reward:     exactReward,
	}

	fmt.Printf("[MINER] Mining Block #%d with %d transactions (Dynamic Difficulty: %d)...\n", newBlock.Index, len(validTx), newBlock.Difficulty)
	
	startTime := time.Now()
	// Execute the CPU-intensive proof-of-work mining algorithm to discover a valid nonce and block hash.
	nonce, hash := lc.Engine.Mine(newBlock)
	duration := time.Since(startTime)

	newBlock.Nonce = nonce
	newBlock.Hash = hash
	newBlock.Reward = exactReward // Enforce correct calculated reward value respecting max supply.

	// --- RIGID CONSENSUS VALIDATION PIPELINE ---
	if err := core.ValidateBlockConsensus(newBlock, parent, currentSupply); err != nil {
		panic(fmt.Sprintf("[CRITICAL CONSENSUS VIOLATION] Generated block failed strict validation: %v", err))
	}
	// -------------------------------------------

	lc.Mu.Lock()
	// Append the successfully mined block to the local chain array and commit it to storage.
	lc.Chain = append(lc.Chain, newBlock)
	lc.Storage.SaveBlock(newBlock.Index, newBlock)

	totalMinerReward := newBlock.Reward + feeTotal
	// Credit the combined block reward and collected transaction fees to the miner account state.
	if totalMinerReward > 0 {
		if _, ok := lc.State[lc.MinerAddress]; !ok {
			lc.State[lc.MinerAddress] = &AccountState{Balance: totalMinerReward, Nonce: 0}
		} else {
			lc.State[lc.MinerAddress].Balance += totalMinerReward
		}
	}
	lc.Mu.Unlock()

	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("[SUCCESS] Block #%d Mined & Saved! (Reward: %.8f, Fee: %.8f, Nonce: %d, Time: %v)\n", newBlock.Index, formatCoin(newBlock.Reward), formatCoin(feeTotal), newBlock.Nonce, duration)
	fmt.Printf("[CHAIN] Total Blocks: %d | Circulating Supply: %.8f / %.8f\n", len(lc.Chain), formatCoin(lc.GetTotalCirculatingSupply()), formatCoin(consensus.MaxSupply))
	fmt.Println("--------------------------------------------------------------------------------")
}
