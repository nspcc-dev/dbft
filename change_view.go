package dbft

// ChangeView represents dBFT ChangeView message.
type ChangeView interface {
	// NewViewNumber returns proposed view number.
	NewViewNumber() byte

	// Reason returns change view reason.
	Reason() ChangeViewReason
}
