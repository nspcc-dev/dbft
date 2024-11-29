package dbft

import (
	"fmt"
)

type (
	// PublicKey is a generic public key interface used by dbft.
	PublicKey any

	// PrivateKey is a generic private key interface used by dbft. PrivateKey is used
	// only by [PreBlock] and [Block] signing callbacks ([PreBlock.SetData] and
	// [Block.Sign]) to grant access to the private key abstraction to Block and
	// PreBlock signing code. PrivateKey does not contain any methods, hence user
	// supposed to perform type assertion before the PrivateKey usage.
	PrivateKey any

	// Hash is a generic hash interface used by dbft for payloads, blocks and
	// transactions identification. It is recommended to implement this interface
	// using hash functions with low hash collision probability. The following
	// requirements must be met:
	// 1. Hashes of two equal payloads/blocks/transactions are equal.
	// 2. Hashes of two different payloads/blocks/transactions are different.
	Hash interface {
		comparable
		fmt.Stringer
	}
)
