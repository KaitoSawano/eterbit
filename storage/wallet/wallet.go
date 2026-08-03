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

package wallet

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"eterbit/crypto"
	"github.com/cloudflare/circl/sign/dilithium/mode3"
)

// WalletData defines the structural schema for serializing cryptographic keypairs and address mappings.
type WalletData struct {
	Address    string `json:"address"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

const walletDir = "eterbit_data"

// SaveWalletCustom serializes wallet credential parameters and commits them to a custom file path.
func SaveWalletCustom(filePath, addr, pubHex, privHex string) error {
	// Extract the directory path portion from the target file path specification.
	dir := filepath.Dir(filePath)
	
	// Ensure that all parent directories leading to the destination file path are created with proper permissions.
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Construct the wallet data structure containing the address and hex-encoded key representations.
	w := WalletData{
		Address:    addr,
		PublicKey:  pubHex,
		PrivateKey: privHex,
	}

	// Serialize the wallet data struct into an indented JSON byte sequence for readability and storage.
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}

	// Write the serialized wallet configuration file to disk with strict file permissions (owner-read/write only).
	return os.WriteFile(filePath, data, 0600)
}

// LoadWalletCustom reads and deserializes a keystore mapping from a specific custom file path.
func LoadWalletCustom(filePath string) (string, *mode3.PrivateKey, []byte, error) {
	// Read the raw file content from the specified keystore file path.
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", nil, nil, err
	}

	var w WalletData
	// Unmarshal the JSON byte payload back into the WalletData structure.
	if err := json.Unmarshal(data, &w); err != nil {
		return "", nil, nil, err
	}

	// Decode the hexadecimal private and public key strings back into raw byte slices.
	privBytes, _ := hex.DecodeString(w.PrivateKey)
	pubBytes, _ := hex.DecodeString(w.PublicKey)

	var priv mode3.PrivateKey
	// Unmarshal the raw private key bytes into a functional post-quantum Dilithium private key instance.
	if err := priv.UnmarshalBinary(privBytes); err != nil {
		return "", nil, nil, err
	}

	// Return the parsed wallet address, private key pointer, public key bytes, and nil error status.
	return w.Address, &priv, pubBytes, nil
}

// SaveWallet serializes the wallet credential parameters to the default keystore file.
func SaveWallet(addr, pubHex, privHex string) error {
	// Delegate the save operation to SaveWalletCustom targeting the default keystore location.
	return SaveWalletCustom(filepath.Join(walletDir, "keystore.json"), addr, pubHex, privHex)
}

// LoadWallet attempts to read and deserialize the default localized keystore mapping from disk storage.
func LoadWallet() (*WalletData, error) {
	filePath := filepath.Join(walletDir, "keystore.json")
	// Read the raw contents of the default keystore file from disk.
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var w WalletData
	// Parse the retrieved JSON data into a WalletData struct instance.
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, err
	}

	// Return a pointer to the loaded wallet data structure and nil error.
	return &w, nil
}

// CreateOrLoadWallet checks for an existing default keystore file. If found, it loads and decodes the keys;
// otherwise, it provisions a new post-quantum Dilithium keypair and persists it.
func CreateOrLoadWallet() (string, *mode3.PrivateKey, []byte, error) {
	// Delegate execution to CreateOrLoadWalletCustom using the default keystore path.
	return CreateOrLoadWalletCustom(filepath.Join(walletDir, "keystore.json"))
}

// CreateOrLoadWalletCustom provisions a new post-quantum Dilithium keypair and persists it to a custom file path.
func CreateOrLoadWalletCustom(filePath string) (string, *mode3.PrivateKey, []byte, error) {
	// Generate a fresh post-quantum cryptographic public and private keypair.
	pub, priv, err := crypto.GenerateKey()
	if err != nil {
		return "", nil, nil, err
	}

	// Marshal the public key instance into its raw binary byte representation.
	pubBytes, err := pub.MarshalBinary()
	if err != nil {
		return "", nil, nil, err
	}
	
	// Marshal the private key instance into its raw binary byte representation.
	privBytes, err := priv.MarshalBinary()
	if err != nil {
		return "", nil, nil, err
	}

	// Derive the blockchain network address string from the raw public key bytes.
	addr := crypto.PubkeyToAddress(pubBytes)
	// Encode both public and private key bytes into hexadecimal strings for persistent storage.
	pubHex := hex.EncodeToString(pubBytes)
	privHex := hex.EncodeToString(privBytes)

	// Persist the newly generated key credentials and address to the specified file path.
	_ = SaveWalletCustom(filePath, addr, pubHex, privHex)

	// Return the derived address, private key pointer, public key bytes, and nil error status.
	return addr, priv, pubBytes, nil
}
