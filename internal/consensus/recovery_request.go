package consensus

import (
	"encoding/gob"

	"github.com/nspcc-dev/dbft"
)

type (
	recoveryRequest struct {
		timestamp uint32
	}
	// recoveryRequestAux is an auxiliary structure for recoveryRequest encoding.
	recoveryRequestAux struct {
		Timestamp uint32
	}
)

var _ dbft.RecoveryRequest = (*recoveryRequest)(nil)

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
