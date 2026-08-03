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

// ConsensusParameters defines the fixed macroeconomic and mathematical rules for the Eterbit blockchain.
type ConsensusParameters struct {
	DifficultyBits uint64 // Target difficulty prefix/bits for Proof-of-Work
	BlockReward    uint64 // Initial mining reward per block (e.g., 50 coins)
	MaxSupply      uint64 // Maximum cap for token issuance
}

// DefaultConsensus returns the standard operational consensus rules for Eterbit.
func DefaultConsensus() *ConsensusParameters {
	return &ConsensusParameters{
		DifficultyBits: 3,
		BlockReward:    50,
		MaxSupply:      785000000, // Adjusting to Eterbit parameters
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
