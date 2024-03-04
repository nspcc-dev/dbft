package dbft

// RecoveryRequest represents dBFT RecoveryRequest message.
type RecoveryRequest interface {
	// Timestamp returns this message's timestamp.
	Timestamp() uint64
	// SetTimestamp sets this message's timestamp.
	SetTimestamp(ts uint64)
}
