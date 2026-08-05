// Copyright (c) 2026 AldianOkto. All rights reserved.
// Copyright (c) 2026 Eterbit Core.
// Use of this source code is governed by the Apache License.
// that can be found in the root directory of this repository.
// Project: Eterbit / Blockchain Core
//
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at <http://www.apache.org/licenses/LICENSE-2.0>
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

// PoWLimit defines the maximum target difficulty limit in Big Integer 256-bit representation.
// Di-loose/longgarkan sementara agar mining genesis block instan di HP/perangkat lokal.
var PoWLimit, _ = new(big.Int).SetString("00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)

// Immutable Network & Macroeconomic Constants (Hardcoded Rules - Cannot be altered arbitrarily)
const (
	CoinUnit           uint64 = 100000000             // 8 Decimals precision factor
	MaxSupply          uint64 = 785000000 * CoinUnit // Fixed Maximum Cap: 785 Million Units
	BlockReward        uint64 = 50 * CoinUnit         // Initial Minting Reward: 50 Units per Block
	HalvingInterval    uint64 = 7850000               // Strict Halving Block Interval
	DefaultPort        int    = 19333                 // Default P2P Network Port
	AddressPrefix      string = "etrb"                // Immutable Wallet Address Prefix
	GenesisBits        uint32 = 0x1e0ffff0            // Compact difficulty bits representation ala Bitcoin Core

	// Proof-of-Work Target Parameters
	PowTargetTimespan  int64  = 2 * 24 * 60 * 60 // Difficulty adjustment span (e.g., 2 Days)
	PowTargetSpacing   int64  = 35               // Target block time spacing in seconds
	TargetBlockTimeSec int64  = PowTargetSpacing // Backward compatibility alias for target block time

	// ExpectedGenesisHash stores the immutable hardcoded SHA3-512 hash checkpoint of the Eterbit Genesis block.
	ExpectedGenesisHash string = "000e409e9ba9cc44032bf91fb345c10817dacc1d9234782d08873cf9b18bb67f803691b65fdc256678b8179fd2939e3c66874e7a5775945df0f42e3652e42c2d"
)

// CheckpointData represents a hardcoded historical block height and its immutable cryptographic hash checkpoint.
type CheckpointData struct {
	Height uint64
	Hash   string
}

// HardcodedCheckpoints stores trusted historical checkpoints to prevent history rewrites and tampering.
var HardcodedCheckpoints = map[uint64]string{
	0: ExpectedGenesisHash,
	// Add future trusted block checkpoints here as the network grows:
	// 10000: "some_block_hash_hex...",
}

// ConsensusParameters defines the fixed macroeconomic and mathematical rules for the Eterbit ledger.
type ConsensusParameters struct {
	DifficultyBits    uint64 // Target difficulty level / factor
	GenesisBits       uint32 // Compact difficulty bits representation
	BlockReward       uint64 // Initial minting reward per block (with 8 decimals precision)
	MaxSupply         uint64 // Maximum cap for token issuance (with 8 decimals precision)
	HalvingInterval   uint64 // Interval blocks for halving
	DefaultPort       int    // Hardcoded network port
	AddressPrefix     string // Hardcoded wallet address prefix
	PowTargetTimespan int64  // Difficulty adjustment timespan
	PowTargetSpacing  int64  // Target block time spacing
}

// DefaultConsensus returns the standard operational consensus rules for Eterbit using PoWLimit baseline.
func DefaultConsensus() *ConsensusParameters {
	return &ConsensusParameters{
		DifficultyBits:    1,               // Initial baseline factor multiplier
		GenesisBits:       GenesisBits,   // Compact target bits
		BlockReward:       BlockReward,
		MaxSupply:         MaxSupply,
		HalvingInterval:   HalvingInterval,
		DefaultPort:       DefaultPort,
		AddressPrefix:     AddressPrefix,
		PowTargetTimespan: PowTargetTimespan,
		PowTargetSpacing:  PowTargetSpacing,
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

// CalculateNextDifficulty implements a dynamic difficulty adjustment
// based on the time taken to process the previous block compared to TargetBlockTimeSec.
func CalculateNextDifficulty(prevBlockTimestamp int64, currentBlockTimestamp int64, prevDifficulty uint64) uint64 {
	if prevBlockTimestamp == 0 || currentBlockTimestamp <= prevBlockTimestamp {
		if prevDifficulty < 1 {
			return 1
		}
		return prevDifficulty
	}

	timeElapsed := currentBlockTimestamp - prevBlockTimestamp
	var newDifficulty uint64 = prevDifficulty

	if timeElapsed < TargetBlockTimeSec {
		newDifficulty = prevDifficulty + 1
	} else if timeElapsed > TargetBlockTimeSec*2 {
		if prevDifficulty > 1 {
			newDifficulty = prevDifficulty - 1
		}
	}

	if newDifficulty < 1 {
		return 1
	}

	return newDifficulty
}

// ValidatePoW verifies whether a given block header hash satisfies the target difficulty limit using pure Big Integer evaluation.
func ValidatePoW(blockHashHex string, difficultyBits uint64) bool {
	hashInt := new(big.Int)
	hashBytes, err := hex.DecodeString(blockHashHex)
	if err != nil {
		return false
	}
	hashInt.SetBytes(hashBytes)

	// Hash must not exceed the absolute PoWLimit threshold
	if hashInt.Cmp(PoWLimit) > 0 {
		return false
	}

	// Derive target difficulty by shifting PoWLimit based on difficulty bits factor
	target := new(big.Int).Set(PoWLimit)
	if difficultyBits > 1 {
		target.Rsh(target, uint(difficultyBits))
	}

	// Valid if hashInt <= target
	return hashInt.Cmp(target) <= 0
}

// ComputeHeaderHash calculates the cryptographic SHA3-512 hash representation for block validation, including the optional genesis message.
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

// VerifyCheckpoint validates any given block height and its hash against the hardcoded historical checkpoints map.
func VerifyCheckpoint(height uint64, blockHash []byte) error {
	expectedHash, exists := HardcodedCheckpoints[height]
	if !exists {
		return nil // No checkpoint defined for this height, skip verification safely
	}

	actualHashHex := hex.EncodeToString(blockHash)
	if actualHashHex != expectedHash {
		return fmt.Errorf("CONSENSUS REJECTION: Checkpoint mismatch at block height %d! Expected '%s', got '%s'. Chain rejected.", height, expectedHash, actualHashHex)
	}
	return nil
}

// VerifyBlockReward checks if the distributed block reward and transaction fees adhere to protocol limits.
func VerifyBlockReward(rewardClaimed uint64, feesCollected uint64, standardReward uint64) bool {
	return rewardClaimed <= (standardReward + feesCollected)
}
