package dbft

// PrepareResponse represents dBFT PrepareResponse message.
type PrepareResponse[H Hash] interface {
	// PreparationHash returns the hash of PrepareRequest payload
	// for this epoch.
	PreparationHash() H
}
