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

package core

import (
	"bytes"
	"encoding/hex"
	"strconv"

	"eterbit/internal/consensus"
	"golang.org/x/crypto/sha3"
)

// CoinUnit defines an 8-decimal scaling factor consistent across all node operations.
const CoinUnit = uint64(100000000)

// Retrieve the maximum total coin supply limit from centralized consensus parameters.
var consensusParams = consensus.DefaultConsensus()

const MaxEterbitSupply uint64 = 785000000 // Aligns with MaxSupply specified in internal/consensus

// LedgerBlock represents the core structural block entity containing transactional ledger data, cryptographic hashes, and consensus metadata.
type LedgerBlock struct {
	Index      uint64     `json:"index"`
	Timestamp  int64      `json:"timestamp"`
	PrevHash   []byte     `json:"prev_hash"`
	Hash       []byte     `json:"hash"`
	Transfers  []*Transfer `json:"transfers"`
	Miner      string     `json:"miner"`
	Nonce      uint64     `json:"nonce"`
	Difficulty uint32     `json:"difficulty"`
	Reward     uint64     `json:"reward"`
}

// GetBlockReward dynamically computes the block reward per block and multiplies it by CoinUnit to maintain 8-decimal accuracy.
func GetBlockReward(blockHeight uint64) uint64 {
	// Multiply the base block reward by CoinUnit to convert it into the smallest fractional integer units (e.g., 50 * 100,000,000).
	initialReward := consensusParams.BlockReward * CoinUnit
	halvingInterval := uint64(7850000) // Halving interval configuration aligned to 7,850,000 blocks

	halvings := blockHeight / halvingInterval

	// Prevent integer overflow or underflow by capping the maximum halving bitwise shifts at 64.
	if halvings >= 64 {
		return 0
	}

	// Apply bitwise right-shift operations to reduce the initial reward by half for each elapsed interval.
	return initialReward >> halvings
}

// ConsensusEngine coordinates the proof-of-work mining process, hash target difficulty evaluation, and block validation parameters.
type ConsensusEngine struct {
	TargetDifficulty uint32
}

// NewConsensusEngine initializes and returns a new ConsensusEngine instance configured with the specified difficulty target.
func NewConsensusEngine(difficulty uint32) *ConsensusEngine {
	// Instantiate and return a ConsensusEngine pointer with the targeted difficulty level.
	return &ConsensusEngine{TargetDifficulty: difficulty}
}

// AssembleBlockData serializes and concatenates block headers, transactional payloads, and a candidate nonce into a unified byte array for hashing.
func (ce *ConsensusEngine) AssembleBlockData(b *LedgerBlock, nonce uint64) []byte {
	var rawTxData []byte
	// Concatenate all transfer signatures included in the block payload.
	for _, tx := range b.Transfers {
		rawTxData = append(rawTxData, tx.Signature...)
	}
	// Join all block components into a single canonical byte array representation.
	return bytes.Join([][]byte{
		b.PrevHash,
		rawTxData,
		[]byte(strconv.FormatUint(b.Index, 16)),
		[]byte(strconv.FormatInt(b.Timestamp, 16)),
		[]byte(strconv.FormatUint(nonce, 16)),
	}, []byte{})
}

// Mine executes an iterative proof-of-work search loop, testing candidate nonces until a hash meeting the target difficulty is discovered.
func (ce *ConsensusEngine) Mine(b *LedgerBlock) (uint64, []byte) {
	// Compute and assign the accurate block reward based on the current block height index.
	b.Reward = GetBlockReward(b.Index)

	var nonce uint64 = 0
	hasher := sha3.New256()

	// Iteratively test nonces until a resulting hash satisfies the proof-of-work difficulty requirement.
	for {
		data := ce.AssembleBlockData(b, nonce)
		hasher.Reset()
		hasher.Write(data)
		hash := hasher.Sum(nil)

		if ce.validateHash(hash) {
			return nonce, hash
		}
		nonce++
	}
}

// validateHash validates the generated hash using validation rules defined within internal/consensus.
func (ce *ConsensusEngine) validateHash(hash []byte) bool {
	hashStr := hex.EncodeToString(hash)
	// Directly invoke the ValidatePoW function provided by the internal/consensus package.
	return consensus.ValidatePoW(hashStr, uint64(ce.TargetDifficulty))
}
