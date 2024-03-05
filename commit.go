package dbft

// Commit is an interface for dBFT Commit message.
type Commit interface {
	// Signature returns commit's signature field
	// which is a block signature for the current epoch.
	Signature() []byte
}
