Eterbit v0.01

Copyright (c) 2026 Eterbit Core.
Copyright (c) 2026 AldianOkto (Subang, Indonesian). All rights reserved.
Distributed under the Apache License, Version 2.0, see the accompanying
file LICENSE or http://www.apache.org/licenses/LICENSE-2.0.
This product includes post-quantum cryptographic software utilizing
Cloudflare CIRCL (Dilithium Mode 3).


Introduction
-----
Eterbit Core is an experimental cryptocurrency built in Go, inspired by the early days of decentralized peer-to-peer digital cash. It utilizes Dilithium signatures for advanced cryptography and a Proof-of-Work (PoW) consensus mechanism.


Set up
-----
Clone the repository and build the binary using Go:

  go build -o eterbit


To support the network by running a mining node, enable the mining routine
and keep the program open. Your computer will be solving computational problems 
using SHA3-256 and Dilithium signatures that are used to lock in blocks of 
transactions. As a reward for supporting the network, you receive newly 
minted coins when you successfully generate a block.
