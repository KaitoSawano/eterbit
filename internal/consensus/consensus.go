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

package consensus

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
)

// Immutable Network & Macroeconomic Constants (Hardcoded Rules - Cannot be altered arbitrarily)
const (
	CoinUnit        uint64 = 100000000          // 8 Decimals precision factor
	MaxSupply       uint64 = 785000000 * CoinUnit // Fixed Maximum Cap: 785 Million Coins
	BlockReward     uint64 = 50 * CoinUnit      // Initial Mining Reward: 50 Coins per Block
	HalvingInterval uint64 = 7850000            // Strict Halving Block Interval
	DefaultPort     int    = 19666              // Default P2P Network Port
	DifficultyBits  uint64 = 3                  // Target Difficulty Prefix/Bits
)

// ConsensusParameters defines the fixed macroeconomic and mathematical rules for the Eterbit blockchain.
type ConsensusParameters struct {
	DifficultyBits  uint64 // Target difficulty prefix/bits for Proof-of-Work
	BlockReward     uint64 // Initial mining reward per block (with 8 decimals precision)
	MaxSupply       uint64 // Maximum cap for token issuance (with 8 decimals precision)
	HalvingInterval uint64 // Interval blocks for halving
	DefaultPort     int    // Hardcoded network port
}

// DefaultConsensus returns the standard operational consensus rules for Eterbit.
func DefaultConsensus() *ConsensusParameters {
	return &ConsensusParameters{
		DifficultyBits:  DifficultyBits,
		BlockReward:     BlockReward,
		MaxSupply:       MaxSupply,
		HalvingInterval: HalvingInterval,
		DefaultPort:     DefaultPort,
	}
}

// ValidatePoW verifies whether a given block header hash satisfies the target difficulty requirement.
func ValidatePoW(blockHash string, difficulty uint64) bool {
	target := createTargetPrefix(difficulty)
	return len(blockHash) >= int(difficulty) && blockHash[:int(difficulty)] == target
}

// ComputeHeaderHash calculates the cryptographic SHA-256 hash representation for block validation.
func ComputeHeaderHash(prevHash string, merkleRoot string, timestamp int64, nonce uint64) string {
	record := bytes.Join([][]byte{
		[]byte(prevHash),
		[]byte(merkleRoot),
		big.NewInt(timestamp).Bytes(),
		big.NewInt(int64(nonce)).Bytes(),
	}, []byte{})

	hash := sha256.Sum256(record)
	return hex.EncodeToString(hash[:])
}

// createTargetPrefix generates the required leading zero pattern based on the difficulty level.
func createTargetPrefix(difficulty uint64) string {
	prefix := ""
	for i := uint64(0); i < difficulty; i++ {
		prefix += "0"
	}
	return prefix
}

// VerifyBlockReward checks if the distributed block reward and transaction fees adhere to protocol limits.
func VerifyBlockReward(rewardClaimed uint64, feesCollected uint64, standardReward uint64) bool {
	return rewardClaimed <= (standardReward + feesCollected)
}
