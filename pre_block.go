package dbft

// PreBlock is a generic interface for a PreBlock used by anti-MEV dBFT extension.
// It holds a "draft" of block that should be converted to a final block with the
// help of additional data held by PreCommit messages.
type PreBlock[H Hash] interface {
	// Data returns PreBlock's data CNs need to exchange during PreCommit phase.
	// Data represents additional information not related to a final block signature.
	Data() []byte
	// SetData generates and sets PreBlock's data CNs need to exchange during
	// PreCommit phase.
	SetData(key PrivateKey) error
	// Verify checks if data related to PreCommit phase is correct. This method is
	// refined on PreBlock rather than on PreCommit message since PreBlock itself is
	// required for PreCommit's data verification. It's guaranteed that all
	// proposed transactions are collected by the moment of call to Verify.
	Verify(key PublicKey, data []byte) error

	// Transactions returns PreBlock's transaction list. This list may be different
	// comparing to the final set of Block's transactions.
	Transactions() []Transaction[H]
	// SetTransactions sets PreBlock's transaction list. This list may be different
	// comparing to the final set of Block's transactions.
	SetTransactions([]Transaction[H])
}
