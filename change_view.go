package dbft

// ChangeView represents dBFT ChangeView message.
type ChangeView interface {
	// NewViewNumber returns proposed view number.
	NewViewNumber() byte

	// Timestamp returns message's timestamp.
	Timestamp() uint64

	// Reason returns change view reason.
	Reason() ChangeViewReason
}
