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

package main

import (
	"flag"
	"fmt"
	"os"

	"eterbit/internal/cli"
	"eterbit/internal/daemon"

	_ "github.com/cloudflare/circl/sign/dilithium/mode3"
)

// main serves as the primary router entry point for the command-line interface application,
// parsing operational arguments and dispatching execution flows to dedicated module handlers.
func main() {
	// Initialize distinct command-line flag sets for various administrative and operational subcommands.
	walletCreateCmd := flag.NewFlagSet("create", flag.ExitOnError)
	balanceCmd := flag.NewFlagSet("balance", flag.ExitOnError)
	supplyCmd := flag.NewFlagSet("supply", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	nodeCmd := flag.NewFlagSet("node", flag.ExitOnError)
	explorerCmd := flag.NewFlagSet("explorer", flag.ExitOnError)
	mineCmd := flag.NewFlagSet("mine", flag.ExitOnError)
	miningCmd := flag.NewFlagSet("mining", flag.ExitOnError)
	peersCmd := flag.NewFlagSet("peers", flag.ExitOnError)
	feesCmd := flag.NewFlagSet("fees", flag.ExitOnError)
	getBlockHashCmd := flag.NewFlagSet("getblockhash", flag.ExitOnError)
	getBlockCmd := flag.NewFlagSet("getblock", flag.ExitOnError)
	uptimeCmd := flag.NewFlagSet("uptime", flag.ExitOnError)
	getNetTotalsCmd := flag.NewFlagSet("getnettotals", flag.ExitOnError)
	addNodeCmd := flag.NewFlagSet("addnode", flag.ExitOnError)

	// Define specific parameter bindings for individual command flags.
	walletLabel := walletCreateCmd.String("label", "Default Account", "Label description for the new multi-wallet account")

	sendRecipient := sendCmd.String("to", "", "Recipient destination address")
	sendAmount := sendCmd.Uint64("amount", 0, "Transfer value amount")
	sendFee := sendCmd.Uint64("fee", 2, "Transaction fee")
	sendSenderAddr := sendCmd.String("from", "", "Specific sender account address within wallet.dat")

	nodePort := nodeCmd.String("port", ":19333", "P2P listening port for the node")
	nodeConnect := nodeCmd.String("connect", "", "Peer address to connect (e.g., localhost:19333)")

	mineBlocks := mineCmd.Int("blocks", 1, "Number of blocks to generate")
	mineAddress := mineCmd.String("address", "", "Target destination address for block reward (generatetoaddress style)")

	addNodeTarget := addNodeCmd.String("to", "", "Target peer address to add (e.g., localhost:19333)")

	// Validate whether adequate command-line arguments have been provided by the executing operator.
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Switch execution branch based upon the primary subcommand supplied in system arguments.
	switch os.Args[1] {
	case "create":
		// Parse creation flag options and invoke the modular CLI wallet creation handler.
		walletCreateCmd.Parse(os.Args[2:])
		cli.HandleCreateWalletAccount(*walletLabel)
	case "balance":
		// Parse balance check options and invoke the modular CLI balance lookup handler.
		balanceCmd.Parse(os.Args[2:])
		cli.HandleCheckBalance()
	case "supply":
		// Parse supply metrics options and invoke the circulating and max supply inspection handler.
		supplyCmd.Parse(os.Args[2:])
		cli.HandleCheckSupply()
	case "send":
		// Parse transaction transfer parameters and invoke the modular CLI transaction dispatch handler.
		sendCmd.Parse(os.Args[2:])
		cli.HandleSendTx(*sendRecipient, *sendAmount, *sendFee, *sendSenderAddr)
	case "node":
		// Parse node server flag parameters and bootstrap the background P2P validator daemon service.
		nodeCmd.Parse(os.Args[2:])
		daemon.RunNodeDaemon(*nodePort, *nodeConnect)
	case "explorer":
		// Parse block inspection flag options and invoke the modular blockchain explorer handler.
		explorerCmd.Parse(os.Args[2:])
		cli.HandleExploreBlockchain()
	case "mine":
		// Parse block generation parameters and invoke the manual Proof-of-Work mining handler.
		mineCmd.Parse(os.Args[2:])
		cli.HandleManualMine(*mineBlocks, *mineAddress)
	case "mining":
		// Validate custom inline target address arguments for simplified mining commands.
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run eterbit.go mining <target_address>")
			os.Exit(1)
		}
		miningCmd.Parse(os.Args[3:])
		cli.HandleManualMine(1, os.Args[2])
	case "addnode":
		// Validate required peer connection parameters before executing manual addnode registration.
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run eterbit.go addnode <host:port>")
			os.Exit(1)
		}
		addNodeCmd.Parse(os.Args[2:])
		cli.HandleAddNode(*addNodeTarget)
	case "peers":
		// Parse peer diagnostic flags and invoke the connected peer information handler.
		peersCmd.Parse(os.Args[2:])
		cli.HandleCheckPeers()
	case "fees":
		// Parse fee market flag options and invoke the mempool fee market statistics handler.
		feesCmd.Parse(os.Args[2:])
		cli.HandleCheckFees()
	case "uptime":
		// Parse uptime diagnostic parameters and invoke the operational duration tracking handler.
		uptimeCmd.Parse(os.Args[2:])
		cli.HandleCheckUptime()
	case "getnettotals":
		// Parse network traffic flags and invoke the network traffic totals diagnostic handler.
		getNetTotalsCmd.Parse(os.Args[2:])
		cli.HandleGetNetTotals()
	case "getblockhash":
		// Verify that the required block index parameter has been adequately supplied prior to command parsing.
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run eterbit.go getblockhash <block_index>")
			os.Exit(1)
		}
		getBlockHashCmd.Parse(os.Args[2:])
		cli.HandleGetBlockHash(os.Args[2])
	case "getblock":
		// Verify that the required block hash parameter has been adequately supplied prior to command parsing.
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run eterbit.go getblock <block_hash>")
			os.Exit(1)
		}
		getBlockCmd.Parse(os.Args[2:])
		cli.HandleGetBlock(os.Args[2])
	default:
		// Fallback to displaying usage instructions if an unrecognized command subcommand is provided.
		printUsage()
		os.Exit(1)
	}
}

// printUsage outputs the standard command-line manual instructions and available command options to standard output.
func printUsage() {
	// Render comprehensive structural documentation regarding available console commands.
	fmt.Println("================================================================================")
	fmt.Println(" ETERBIT BLOCKCHAIN CLI MANAGER (MULTI-WALLET ARCHITECTURE)")
	fmt.Println("================================================================================")
	fmt.Println("Available commands:")
	fmt.Println("  go run eterbit.go create [-label <account_label>]")
	fmt.Println("  go run eterbit.go balance")
	fmt.Println("  go run eterbit.go supply")
	fmt.Println("  go run eterbit.go send -to <addr> -amount <val> [-fee <val>] [-from <sender_addr>]")
	fmt.Println("  go run eterbit.go node [--port :port] [--connect host:port]")
	fmt.Println("  go run eterbit.go addnode <host:port>")
	fmt.Println("  go run eterbit.go mine [-blocks <num>] [-address <addr>]")
	fmt.Println("  go run eterbit.go mining <target_address>")
	fmt.Println("  go run eterbit.go explorer")
	fmt.Println("  go run eterbit.go peers")
	fmt.Println("  go run eterbit.go fees")
	fmt.Println("  go run eterbit.go uptime")
	fmt.Println("  go run eterbit.go getnettotals")
	fmt.Println("  go run eterbit.go getblockhash <index>")
	fmt.Println("  go run eterbit.go getblock <hash>")
	fmt.Println("================================================================================")
}
