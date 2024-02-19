package dbft

// ChangeView represents dBFT ChangeView message.
type ChangeView interface {
	// NewViewNumber returns proposed view number.
	NewViewNumber() byte

	// SetNewViewNumber sets the proposed view number.
	SetNewViewNumber(view byte)

	// Timestamp returns message's timestamp.
	Timestamp() uint64

	// SetTimestamp sets message's timestamp.
	SetTimestamp(ts uint64)

	// Reason returns change view reason.
	Reason() ChangeViewReason

	// SetReason sets change view reason.
	SetReason(reason ChangeViewReason)
}
