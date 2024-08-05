BTC Node Challenge

This repository contains an implementation of the first three milestones of the challenge described below.
You can build the code with `make build` (or just `go build`) and run it with `./btc-node-challenge`. Tests can be run
with `make test`.

On launch, the program connects to a hard-coded node taken from https://bitnodes.io, performs a protocol handshake and
sends an `getaddr` message to it to discover more peers. It then connects to some of them, trying to maintain at least
ten connections.

The program processes `inv` messages received from the connected nodes and requests blocks contained in those messages.
Received blocks are decoded and stored in memory, ordered by timestamp. Gaps between blocks in that list are filled by
requesting the missing blocks. On graceful shutdown, the program writes the collected blocks to a file called
`state.bin`, which is loaded on subsequent executions.

##### Requirements:
- The implementation should compile at least on linux
- The solution cannot use existing P2P libraries

##### Milestone 1:
- The solution has to perform a full **protocol-level** (post-TCP) handshake with the target node you can choose an available BTC Nodes for connection: https://bitnodes.io/
- You can follow the specification here: https://en.bitcoin.it/
- Can not use the node implementation as a dependency
- You can ignore any post-handshake traffic from the target node, and it doesn't have to keep the connection alive.

##### Milestone 2:
- Manage more than 1 connected node
- Need to perform a full **protocol-level** handshake with the new nodes
- You can find the specification here: https://en.bitcoin.it/wiki/Protocol_documentation#addr and https://en.bitcoin.it/wiki/Protocol_documentation#getaddr

##### Millestone 3:
- Request block informations from other peers, can be the full block data or just the headers
- Your node should be resilient to network failures (dial error, protocol not supported, incompatible version)
- Your node should check the response contents and ignore if the response doesn't contains what was requested, as well as to guarantee the chain consistency, the current should be father of the next one and so on and so forth
- No need for block or header validation, just retrieve and store should be enough

##### Milestone 4:
- Starting from the genesis block you must retrieve few blocks and must verify their transactions.
- You can find the specification here: https://en.bitcoin.it/wiki/Protocol_documentation#Transaction_Verification
- Bonus points if you implement your own Script (stack-based scripting system for transactions) validation, you can find the spec here https://en.bitcoin.it/wiki/Script
- The program can exit after validating the blocks, no need to keep syncing.

##### Milestone 5:
- This last milestone is a continuation of the Milestone 4
- The node should be able to keep syncing (retrieving, validating and importing) blocks until the tip of the chain
- The implemented node should be able to gracefully shutdown (preserving the current sync state) as well as able to resume from the latest point.

##### Evaluation
- **Quality**: the solution should be idiomatic and adhere to Go coding conventions.
- **Performance**: the solution should be as fast as the handshake protocol allows, and it shouldn't block resources.
- **Security**: the network is an inherently untrusted environment, and it should be taken into account.
- **Minimalism**: any number of dependencies can be used, but they should be tailored to the task.

