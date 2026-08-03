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
	"sync"
	"time"

	"eterbit/core"
	"eterbit/internal/consensus"
	"eterbit/storage"
)

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
	// Divide the raw integer smallest unit value by the coin unit multiplier to obtain the standard float representation.
	return float64(amount) / float64(CoinUnit)
}

// CalculateBlockReward calculates the block reward per block utilizing a halving mechanism.
func CalculateBlockReward(blockIndex uint64) uint64 {
	// Define the initial block reward allocation amount scaled to coin units.
	initialReward := uint64(50) * CoinUnit
	halvingInterval := uint64(7850000)
	
	// Compute the total number of halving cycles elapsed based on the current block height index.
	halvings := blockIndex / halvingInterval
	
	// Cap the maximum halving shifts to prevent integer underflow or bitwise overflow.
	if halvings >= 64 {
		return 0
	}
	
	// Apply bitwise right-shift operations to reduce the initial reward by half for each elapsed interval.
	return initialReward >> halvings
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

// LoadFromDisk loads existing blockchain blocks from LevelDB disk storage and rebuilds the account state.
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
	return len(lc.Chain) > 0
}

// RebuildState updates the account balances and nonces based on the transactions within a block.
func (lc *LedgerCore) RebuildState(block *core.LedgerBlock) {
	// Iterate through all transfer transactions recorded within the processed block.
	for _, tx := range block.Transfers {
		sender := hex.EncodeToString(tx.SenderPubKey[:16])
		if _, ok := lc.State[sender]; !ok {
			lc.State[sender] = &AccountState{Balance: InitialAirdrop, Nonce: 0}
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
	genesis := &core.LedgerBlock{
		Index:      0,
		Timestamp:  time.Now().Unix(),
		PrevHash:   make([]byte, 32),
		Transfers:  []*core.Transfer{},
		Miner:      "SYSTEM_GENESIS",
		Nonce:      0,
		Difficulty: lc.Engine.TargetDifficulty,
		Reward:     exactReward,
	}
	// Execute the consensus mining algorithm to solve the genesis block proof-of-work puzzle.
	_, genesis.Hash = lc.Engine.Mine(genesis)
	genesis.Reward = exactReward // Protect the genesis reward value against external modifications.

	// Append the newly minted genesis block to the local chain array and persist it to storage.
	lc.Chain = append(lc.Chain, genesis)
	lc.Storage.SaveBlock(0, genesis)
}

// AddToMempool validates and inserts a transaction payload into the pending mempool queue with Fee Market priority sorting.
func (lc *LedgerCore) AddToMempool(tx *core.Transfer) bool {
	lc.Mu.Lock()
	defer lc.Mu.Unlock()

	// Verify the cryptographic signature authenticity of the incoming transfer transaction.
	if !tx.Verify() {
		fmt.Println("[MEMPOOL] Invalid transaction cryptographic signature!")
		return false
	}

	sender := hex.EncodeToString(tx.SenderPubKey[:16])
	acc, exists := lc.State[sender]
	
	// Initialize account states if missing, validating balance availability against initial airdrop parameters.
	if !exists {
		lc.State[sender] = &AccountState{Balance: InitialAirdrop, Nonce: tx.Nonce}
		acc = lc.State[sender]
	} else if acc.Balance < (tx.Value + tx.Fee) {
		acc.Balance = InitialAirdrop
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
				lc.State[sender] = &AccountState{Balance: InitialAirdrop, Nonce: 0}
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
	exactReward := CalculateBlockReward(nextIndex)

	// Construct the new ledger block container structure with current parameters.
	newBlock := &core.LedgerBlock{
		Index:      nextIndex,
		Timestamp:  time.Now().Unix(),
		PrevHash:   parent.Hash,
		Transfers:  validTx,
		Miner:      lc.MinerAddress,
		Difficulty: lc.Engine.TargetDifficulty,
		Reward:     exactReward,
	}

	fmt.Printf("[MINER] Mining Block #%d with %d transactions (Difficulty: %d)...\n", newBlock.Index, len(validTx), newBlock.Difficulty)
	
	startTime := time.Now()
	// Execute the CPU-intensive proof-of-work mining algorithm to discover a valid nonce and block hash.
	nonce, hash := lc.Engine.Mine(newBlock)
	duration := time.Since(startTime)

	newBlock.Nonce = nonce
	newBlock.Hash = hash
	newBlock.Reward = exactReward // Enforce the correct halving reward value to prevent accidental overrides during mining.

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
	fmt.Printf("[CHAIN] Total Blocks: %d | Transactions Processed: %d\n", len(lc.Chain), len(validTx))
	fmt.Println("--------------------------------------------------------------------------------")
}
