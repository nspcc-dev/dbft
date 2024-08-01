package dbft

// Commit is an interface for dBFT Commit message.
type Commit interface {
	// Signature returns commit's signature field
	// which is a final block signature for the current epoch for both dBFT 2.0 and
	// for anti-MEV extension.
	Signature() []byte
}
