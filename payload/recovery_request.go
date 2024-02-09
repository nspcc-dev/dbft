package payload

import (
	"encoding/gob"
)

// RecoveryRequest represents dBFT RecoveryRequest message.
type RecoveryRequest interface {
	// Timestamp returns this message's timestamp.
	Timestamp() uint64
	// SetTimestamp sets this message's timestamp.
	SetTimestamp(ts uint64)
}

type (
	recoveryRequest struct {
		timestamp uint32
	}
	// recoveryRequestAux is an auxiliary structure for recoveryRequest encoding.
	recoveryRequestAux struct {
		Timestamp uint32
	}
)

var _ RecoveryRequest = (*recoveryRequest)(nil)

// EncodeBinary implements Serializable interface.
func (m recoveryRequest) EncodeBinary(w *gob.Encoder) error {
	return w.Encode(&recoveryRequestAux{
		Timestamp: m.timestamp,
	})
}

// DecodeBinary implements Serializable interface.
func (m *recoveryRequest) DecodeBinary(r *gob.Decoder) error {
	aux := new(recoveryRequestAux)
	if err := r.Decode(aux); err != nil {
		return err
	}

	m.timestamp = aux.Timestamp
	return nil
}

// Timestamp implements RecoveryRequest interface.
func (m *recoveryRequest) Timestamp() uint64 {
	return secToNanoSec(m.timestamp)
}

// SetTimestamp implements RecoveryRequest interface.
func (m *recoveryRequest) SetTimestamp(ts uint64) {
	m.timestamp = nanoSecToSec(ts)
}
