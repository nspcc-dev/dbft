package payload

import "github.com/nspcc-dev/neo-go/pkg/io"

// RecoveryRequest represents dBFT RecoveryRequest message.
type RecoveryRequest interface {
	// Timestamp returns this message's timestamp.
	Timestamp() uint32
	// SetTimestamp sets this message's timestamp.
	SetTimestamp(ts uint32)
}

type recoveryRequest struct {
	timestamp uint32
}

var _ RecoveryRequest = (*recoveryRequest)(nil)

// EncodeBinary implements io.Serializable interface.
func (m recoveryRequest) EncodeBinary(w *io.BinWriter) {
	w.WriteU32LE(m.timestamp)
}

// DecodeBinary implements io.Serializable interface.
func (m *recoveryRequest) DecodeBinary(r *io.BinReader) {
	m.timestamp = r.ReadU32LE()
}

// Timestamp implements RecoveryRequest interface.
func (m *recoveryRequest) Timestamp() uint32 {
	return m.timestamp
}

// SetTimestamp implements RecoveryRequest interface.
func (m *recoveryRequest) SetTimestamp(ts uint32) {
	m.timestamp = ts
}
