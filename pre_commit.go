package dbft

// PreCommit is an interface for dBFT PreCommit message. This message is used right
// before the Commit phase to exchange additional information required for the final
// block construction in anti-MEV dBFT extension.
type PreCommit interface {
	// Data returns PreCommit's data that should be used for the final
	// Block construction in anti-MEV dBFT extension.
	Data() []byte
}
