package payload

import (
	"encoding/gob"

	"github.com/nspcc-dev/dbft"
)

type (
	changeView struct {
		newViewNumber byte
		timestamp     uint32
	}
	// changeViewAux is an auxiliary structure for changeView encoding.
	changeViewAux struct {
		Timestamp uint32
	}
)

var _ dbft.ChangeView = (*changeView)(nil)

// EncodeBinary implements Serializable interface.
func (c changeView) EncodeBinary(w *gob.Encoder) error {
	return w.Encode(&changeViewAux{
		Timestamp: c.timestamp,
	})
}

// DecodeBinary implements Serializable interface.
func (c *changeView) DecodeBinary(r *gob.Decoder) error {
	aux := new(changeViewAux)
	if err := r.Decode(aux); err != nil {
		return err
	}
	c.timestamp = aux.Timestamp
	return nil
}

// NewViewNumber implements ChangeView interface.
func (c changeView) NewViewNumber() byte {
	return c.newViewNumber
}

// SetNewViewNumber implements ChangeView interface.
func (c *changeView) SetNewViewNumber(view byte) {
	c.newViewNumber = view
}

// Timestamp implements ChangeView interface.
func (c changeView) Timestamp() uint64 {
	return secToNanoSec(c.timestamp)
}

// SetTimestamp implements ChangeView interface.
func (c *changeView) SetTimestamp(ts uint64) {
	c.timestamp = nanoSecToSec(ts)
}

// Reason implements ChangeView interface.
func (c changeView) Reason() dbft.ChangeViewReason {
	return dbft.CVUnknown
}

// SetReason implements ChangeView interface.
func (c *changeView) SetReason(_ dbft.ChangeViewReason) {
}
