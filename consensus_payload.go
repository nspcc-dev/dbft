package dbft

// ConsensusPayload is a generic payload type which is exchanged
// between the nodes.
type ConsensusPayload[H Hash] interface {
	ConsensusMessage[H]

	// ValidatorIndex returns index of validator from which
	// payload was originated from.
	ValidatorIndex() uint16

	// SetValidatorIndex sets validator index.
	SetValidatorIndex(i uint16)

	Height() uint32

	// Hash returns 32-byte checksum of the payload.
	Hash() H
}
