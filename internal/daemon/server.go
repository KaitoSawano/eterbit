// Copyright (c) 2026 AldianOkto. All rights reserved.
// Copyright (c) 2026 Eterbit Core.
// Use of this source code is governed by the Apache License.
// that can be found in the root directory of this repository.
// Project: Eterbit / Blockchain Core

package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"eterbit/core"
	"eterbit/internal"
	"eterbit/internal/cli"
	"eterbit/internal/p2p"
	"eterbit/node"
	"eterbit/storage/wallet"
)

// RunNodeDaemon initiates the continuous background validation daemon process,
// acting as the primary P2P node runner (Bitcoin-like bitcoind daemon architecture).
func RunNodeDaemon(port string, connectPeer string) {
	fmt.Println("[SYS] Booting Eterbit Live Node Daemon (Bitcoin Core Style)...")
	
	// Record the system startup timestamp for precise uptime tracking functionality.
	internal.RecordStartTime()

	// Load the default validator miner wallet credentials from wallet.dat for block reward distribution.
	wf, err := wallet.LoadWallet()
	var addrMiner string
	if err != nil || wf == nil || len(wf.Accounts) == 0 {
		addrMiner = "SYSTEM_MINER"
	} else {
		addrMiner = wf.Accounts[0].Address
	}

	// Initialize database storage directory and state ledger contexts.
	dataDir := cli.GetDataDir()
	ledger := node.InitializeLedger(dataDir, 3, addrMiner)
	server := p2p.NewServer(port)

	// Register NetTotals HTTP endpoint handler on the P2P server HTTP multiplexer.
	internal.RegisterNetTotalsHandler(server.Mux())

	// Spawn a background worker routine to periodically dump connected peer information 
	// and addrman discovered network addresses to external JSON storage files.
	go func() {
		for {
			time.Sleep(2 * time.Second)
			peerList := server.GetPeerList()
			data, _ := json.MarshalIndent(peerList, "", "  ")
			os.WriteFile(filepath.Join(dataDir, "peers.json"), data, 0644)
			
			// Persist known peer addresses managed by AddrManager for inspection.
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
		
		// Synchronize the incoming transaction directly to persistent disk mempool storage.
		diskMempool := cli.LoadMempoolFromDisk()
		diskMempool = append(diskMempool, tx)
		cli.SaveMempoolToDisk(diskMempool)
	}

	// Define the network block reception callback handler for incoming P2P blocks.
	onBlock := func(block *core.LedgerBlock) {
		fmt.Printf("[P2P] Received new block #%d from network peer!\n", block.Index)
	}

	// Start the P2P networking listener server asynchronously in the background.
	go func() {
		if err := server.StartListening(onBlock, onTx); err != nil {
			fmt.Printf("[P2P] Server error: %v\n", err)
		}
	}()

	// Automatically discover and connect to network peers via seeds and addrman database.
	server.AutoDiscoverAndConnect(connectPeer)

	// Launch a continuous Proof-of-Work mining loop daemon to process pending transactions from the mempool.
	go func() {
		for {
			time.Sleep(3 * time.Second)
			diskMempool := cli.LoadMempoolFromDisk()
			if len(diskMempool) > 0 {
				ledger.Mu.Lock()
				ledger.Mempool = diskMempool
				ledger.Mu.Unlock()

				fmt.Println("[NODE] Pending transactions detected in mempool. Starting Proof-of-Work...")
				ledger.MineBlock()
				cli.SaveMempoolToDisk([]*core.Transfer{})
			}
		}
	}()

	// Output operational node status parameters to standard output.
	fmt.Printf("[NODE] Active validator miner: %s\n", addrMiner)
	fmt.Printf("[NODE] P2P Server listening on %s\n", port)
	fmt.Println("[NODE] Node operational and listening. Press Ctrl+C to terminate.")
	
	// Block main execution thread indefinitely to maintain the live daemon process.
	select {}
}
