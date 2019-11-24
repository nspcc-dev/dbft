package block

import (
	"github.com/CityOfZion/neo-go/pkg/util"
)

// Transaction is a generic transaction interface.
type Transaction interface {
	// Hash must return cryptographic hash of the transaction.
	// Transactions which have equal hashes are considered equal.
	Hash() util.Uint256
}
