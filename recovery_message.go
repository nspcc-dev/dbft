package dbft

// RecoveryMessage represents dBFT Recovery message.
type RecoveryMessage[H Hash, A Address] interface {
	// AddPayload adds payload from this epoch to be recovered.
	AddPayload(p ConsensusPayload[H, A])
	// GetPrepareRequest returns PrepareRequest to be processed.
	GetPrepareRequest(p ConsensusPayload[H, A], validators []PublicKey, primary uint16) ConsensusPayload[H, A]
	// GetPrepareResponses returns a slice of PrepareResponse in any order.
	GetPrepareResponses(p ConsensusPayload[H, A], validators []PublicKey) []ConsensusPayload[H, A]
	// GetChangeViews returns a slice of ChangeView in any order.
	GetChangeViews(p ConsensusPayload[H, A], validators []PublicKey) []ConsensusPayload[H, A]
	// GetCommits returns a slice of Commit in any order.
	GetCommits(p ConsensusPayload[H, A], validators []PublicKey) []ConsensusPayload[H, A]

	// PreparationHash returns has of PrepareRequest payload for this epoch.
	// It can be useful in case only PrepareResponse payloads were received.
	PreparationHash() *H
	// SetPreparationHash sets preparation hash.
	SetPreparationHash(h *H)
}
