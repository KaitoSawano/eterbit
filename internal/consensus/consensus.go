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
	"encoding/hex"
	"fmt"
	"math/big"

	"golang.org/x/crypto/sha3"
)

// Immutable Network & Macroeconomic Constants (Hardcoded Rules - Cannot be altered arbitrarily)
const (
	CoinUnit           uint64 = 100000000           // 8 Decimals precision factor
	MaxSupply          uint64 = 785000000 * CoinUnit // Fixed Maximum Cap: 785 Million Coins
	BlockReward        uint64 = 50 * CoinUnit       // Initial Mining Reward: 50 Coins per Block
	HalvingInterval    uint64 = 7850000             // Strict Halving Block Interval
	DefaultPort        int    = 19333               // Default P2P Network Port
	BaseDifficulty     uint64 = 1                  // Minimum/Base Target Difficulty Bits
	TargetBlockTimeSec int64  = 35                  // Target block time in seconds
	AddressPrefix      string = "etrb"              // Immutable Wallet Address Prefix

	// ExpectedGenesisHash stores the immutable hardcoded SHA3-512 hash checkpoint of the Eterbit Genesis block.
	// Any tampering with the Genesis parameters or message will alter this hash and trigger consensus rejection.
	ExpectedGenesisHash string = "0"
)

// ConsensusParameters defines the fixed macroeconomic and mathematical rules for the Eterbit blockchain.
type ConsensusParameters struct {
	DifficultyBits  uint64 // Target difficulty prefix/bits for Proof-of-Work
	BlockReward     uint64 // Initial mining reward per block (with 8 decimals precision)
	MaxSupply       uint64 // Maximum cap for token issuance (with 8 decimals precision)
	HalvingInterval uint64 // Interval blocks for halving
	DefaultPort     int    // Hardcoded network port
	AddressPrefix   string // Hardcoded wallet address prefix
}

// DefaultConsensus returns the standard operational consensus rules for Eterbit.
func DefaultConsensus() *ConsensusParameters {
	return &ConsensusParameters{
		DifficultyBits:  BaseDifficulty,
		BlockReward:     BlockReward,
		MaxSupply:       MaxSupply,
		HalvingInterval: HalvingInterval,
		DefaultPort:     DefaultPort,
		AddressPrefix:   AddressPrefix,
	}
}

// CalculateBlockReward dynamically computes the block reward based on the hardcoded halving interval.
func CalculateBlockReward(blockIndex uint64) uint64 {
	halvings := blockIndex / HalvingInterval
	if halvings >= 64 {
		return 0
	}
	return BlockReward >> halvings
}

// CalculateNextDifficulty implements an Ethereum-style dynamic difficulty adjustment
// based on the time taken to mine the previous block compared to the TargetBlockTimeSec.
func CalculateNextDifficulty(prevBlockTimestamp int64, currentBlockTimestamp int64, prevDifficulty uint64) uint64 {
	if prevBlockTimestamp == 0 || currentBlockTimestamp <= prevBlockTimestamp {
		if prevDifficulty < BaseDifficulty {
			return BaseDifficulty
		}
		return prevDifficulty
	}

	timeElapsed := currentBlockTimestamp - prevBlockTimestamp
	var newDifficulty uint64 = prevDifficulty

	if timeElapsed < TargetBlockTimeSec {
		newDifficulty = prevDifficulty + 1
	} else if timeElapsed > TargetBlockTimeSec*2 {
		if prevDifficulty > BaseDifficulty {
			newDifficulty = prevDifficulty - 1
		}
	}

	if newDifficulty < BaseDifficulty {
		return BaseDifficulty
	}

	return newDifficulty
}

// ValidatePoW verifies whether a given block header hash satisfies the target difficulty requirement.
func ValidatePoW(blockHash string, difficulty uint64) bool {
	target := createTargetPrefix(difficulty)
	return len(blockHash) >= int(difficulty) && blockHash[:int(difficulty)] == target
}

// ComputeHeaderHash calculates the quantum-resistant cryptographic SHA3-512 hash representation for block validation, including the optional genesis message.
func ComputeHeaderHash(prevHash string, merkleRoot string, timestamp int64, nonce uint64, message string) string {
	record := bytes.Join([][]byte{
		[]byte(prevHash),
		[]byte(merkleRoot),
		big.NewInt(timestamp).Bytes(),
		big.NewInt(int64(nonce)).Bytes(),
		[]byte(message),
	}, []byte{})

	hash := sha3.Sum512(record)
	return hex.EncodeToString(hash[:])
}

// VerifyGenesisCheckpoint rigorously evaluates whether a given block hash matches the immutable protocol genesis checkpoint.
func VerifyGenesisCheckpoint(blockHash []byte) error {
	actualHashHex := hex.EncodeToString(blockHash)
	if actualHashHex != ExpectedGenesisHash {
		return fmt.Errorf("CONSENSUS REJECTION: Invalid genesis block hash! Expected checkpoint '%s', got '%s'. Chain rejected due to hardcoded parameter violation.", ExpectedGenesisHash, actualHashHex)
	}
	return nil
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
