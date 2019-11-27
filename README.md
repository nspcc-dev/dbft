# DBFT
This repo contains Go implementation of the dBFT 2.0 consensus algorithm.

## Design and structure
1. All control flow is done in main package. Most of the code which communicates with external
world (event time events) is hidden behind interfaces and callbacks. As a consequence it is
highly flexible and extendable. Description of config options can be found in `config.go`.
2. `crypto` package contains `PrivateKey`/`PublicKey` interfaces which permits usage of one's own
cryptography for signing blocks on `Commit` stage.
Default implementation with ECDSA signatures is provided, BLS multisignatures could be added
in the nearest future.
3. `block` package contains `Block` and `Transaction` abstractions.
Every block must be able to be signed and verified as well as
implement setters and getters for main fields. Minimal default implementation is provided.
`Transaction` is an entity which can be hashed. Two transactions having equal hashes are considered
equal.
4. `payload` contains interfaces for payloads and minimal implementations. Note that
default implementations do not contain any signatures, so you must wrap them or implement your
own payloads in order to sign and verify messages.
5. `timer` contains default time provider. It should make it easier to write tests
concerning dBFT's time depending behaviour.
6. `simulation` contains an example of dBFT's usage with 6-node consensus. 

## Usage
A client of the library must implement its own event loop.
The library provides 4 callbacks:
- `Start()` which initializes internal dBFT structures
- `OnTransaction()` which must be called everytime new transaction appears
- `OnReceive()` which must be called everytime new payload is received
- `OnTimer()` which must be called everytime timer fires

A minimal example can be found in `simulation/main.go`.

## Links
- dBFT high-level description on NEO website [https://docs.neo.org/docs/en-us/tooldev/concept/consensus/consensus_algorithm.html](https://docs.neo.org/docs/en-us/tooldev/concept/consensus/consensus_algorithm.html)
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
3. `ProcessBlock` is a callback which is called synchronously everytime new block is accepted.
It can or can not persist block but it MUST update all blockchain state
so that other callbacks including `CurrentHeight` and `CurrentHash` return new values.
