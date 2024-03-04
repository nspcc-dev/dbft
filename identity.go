package dbft

import (
	"encoding"
	"fmt"
)

type (
	// PublicKey is a generic public key interface used by dbft.
	PublicKey interface {
		encoding.BinaryMarshaler
		encoding.BinaryUnmarshaler

		// Verify verifies if sig is indeed msg's signature.
		Verify(msg, sig []byte) error
	}

	// PrivateKey is a generic private key interface used by dbft.
	PrivateKey interface {
		// Sign returns msg's signature and error on failure.
		Sign(msg []byte) (sig []byte, err error)
	}

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

	// Address is a generic address interface used by dbft for operations related
	// to consensus address. It is recommended to implement this interface
	// using hash functions with low hash collision probability. The following
	// requirements must be met:
	// 1. Addresses of two equal sets of consensus members are equal.
	// 2. Addresses of two different sets of consensus members are different.
	Address interface {
		comparable
	}
)
