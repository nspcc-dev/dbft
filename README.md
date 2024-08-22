[![Go Reference](https://pkg.go.dev/badge/github.com/nspcc-dev/dbft.svg)](https://pkg.go.dev/github.com/nspcc-dev/dbft/)
![Codecov](https://img.shields.io/codecov/c/github/nspcc-dev/dbft.svg)
[![Report](https://goreportcard.com/badge/github.com/nspcc-dev/dbft)](https://goreportcard.com/report/github.com/nspcc-dev/dbft)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/nspcc-dev/dbft?sort=semver)
![License](https://img.shields.io/github/license/nspcc-dev/dbft.svg?style=popout)

# DBFT
This repo contains Go implementation of the dBFT 2.0 consensus algorithm and its models
written in [TLA⁺](https://lamport.azurewebsites.net/tla/tla.html) language.

## Design and structure
1. All control flow is done in main `dbft` package. Most of the code which communicates with external
world (event time events) is hidden behind interfaces, callbacks and generic parameters. As a
consequence it is highly flexible and extendable. Description of config options can be found
in `config.go`.
2. `dbft` package contains `PrivateKey`/`PublicKey` interfaces which permits usage of one's own
cryptography for signing blocks on `Commit` stage. Refer to `identity.go` for `PrivateKey`/`PublicKey`
description. No default implementation is provided.
3. `dbft` package contains `Hash` interface which permits usage of one's own
hash implementation without additional overhead on conversions. Instantiate dBFT with
custom hash implementation that matches requirements specified in the corresponding
documentation. Refer to `identity.go` for `Hash` description. No default implementation is
provided.
4. `dbft` package contains `Block` and `Transaction` abstractions located at the `block.go` and
`transaction.go` files. Every block must be able to be signed and verified as well as implement getters
for main fields. `Transaction` is an entity which can be hashed. Two entities having
equal hashes are considered equal. No default implementation is provided.
5. `dbft` contains generic interfaces for payloads. No default implementation is provided.
6. `dbft` contains generic `Timer` interface for time-related operations. `timer` package contains
default `Timer` provider that can safely be used in production code. The interface itself
is mostly created for tests dealing with dBFT's time-dependant behaviour.
7. `internal` contains an example of custom identity types and payloads implementation used to implement
an example of dBFT's usage with 6-node consensus. Refer to `internal` subpackages for type-specific dBFT
implementation and tests. Refer to `internal/simulation` for an example of dBFT library usage.
8. `formal-models` contains the set of dBFT's models written in [TLA⁺](https://lamport.azurewebsites.net/tla/tla.html)
language and instructions on how to run and check them. Please, refer to the [README](./formal-models/README.md)
for more details.

## Usage
A client of the library must implement its own event loop.
The library provides 5 callbacks that change the state of the consensus
process:
- `Start()` which initializes internal dBFT structures
- `Reset()` which reinitializes the consensus process
- `OnTransaction()` which must be called everytime new transaction appears
- `OnReceive()` which must be called everytime new payload is received
- `OnTimer()` which must be called everytime timer fires

A minimal example can be found in `internal/simulation/main.go`.

## Links
- dBFT high-level description on NEO website [https://docs.neo.org/docs/en-us/basic/consensus/dbft.html](https://docs.neo.org/docs/en-us/basic/consensus/dbft.html)
- dBFT research paper [https://github.com/NeoResearch/yellowpaper/blob/master/releases/08_dBFT.pdf](https://github.com/NeoResearch/yellowpaper/blob/master/releases/08_dBFT.pdf)

## Notes
1. C# NEO node implementation works with the memory pool model, where only transaction hashes
are proposed in the first step of the consensus and
transactions are synchronized in the background.
Some of the callbacks are in config with sole purpose to support this usecase. However it is 
very easy to extend `PrepareRequest` to also include proposed transactions.
2. NEO has the ability to change the list nodes which verify the block (they are called Validators). This is done through `GetValidators`
callback which is called at the start of every epoch. In the simple case where validators are constant
it can return the same value everytime it is called.
3. `ProcessBlock` is a callback which is called synchronously every time new block is accepted.
It can or can not persist block; it also may keep the blockchain state unchanged. dBFT will NOT
be initialized at the next height by itself to collect the next block until `Reset`
is called. In other words, it's the caller's responsibility to initialize dBFT at the next height even
after block collection at the current height. It's also the caller's responsibility to update the
blockchain state before the next height initialization so that other callbacks including
`CurrentHeight` and `CurrentHash` return new values.
