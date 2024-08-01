package dbft

// RecoveryMessage represents dBFT Recovery message.
type RecoveryMessage[H Hash] interface {
	// AddPayload adds payload from this epoch to be recovered.
	AddPayload(p ConsensusPayload[H])
	// GetPrepareRequest returns PrepareRequest to be processed.
	GetPrepareRequest(p ConsensusPayload[H], validators []PublicKey, primary uint16) ConsensusPayload[H]
	// GetPrepareResponses returns a slice of PrepareResponse in any order.
	GetPrepareResponses(p ConsensusPayload[H], validators []PublicKey) []ConsensusPayload[H]
	// GetChangeViews returns a slice of ChangeView in any order.
	GetChangeViews(p ConsensusPayload[H], validators []PublicKey) []ConsensusPayload[H]
	// GetPreCommits returns a slice of PreCommit messages in any order.
	// If implemented on networks with no AntiMEV extension it can just
	// always return nil.
	GetPreCommits(p ConsensusPayload[H], validators []PublicKey) []ConsensusPayload[H]
	// GetCommits returns a slice of Commit in any order.
	GetCommits(p ConsensusPayload[H], validators []PublicKey) []ConsensusPayload[H]

	// PreparationHash returns has of PrepareRequest payload for this epoch.
	// It can be useful in case only PrepareResponse payloads were received.
	PreparationHash() *H
}
