package payload

import "github.com/CityOfZion/neo-go/pkg/io"

// ChangeView represents dBFT ChangeView message.
type ChangeView interface {
	// NewViewNumber returns proposed view number.
	NewViewNumber() byte

	// SetNewViewNumber sets the proposed view number.
	SetNewViewNumber(view byte)

	// Timestamp returns message's timestamp.
	Timestamp() uint32

	// SetTimestamp sets message's timestamp.
	SetTimestamp(ts uint32)
}

type changeView struct {
	newViewNumber byte
	timestamp     uint32
}

var _ ChangeView = (*changeView)(nil)

// EncodeBinary implements io.Serializable interface.
func (c changeView) EncodeBinary(w *io.BinWriter) {
	w.WriteLE(c.timestamp)
}

// DecodeBinary implements io.Serializable interface.
func (c *changeView) DecodeBinary(r *io.BinReader) {
	r.ReadLE(&c.timestamp)
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
func (c changeView) Timestamp() uint32 {
	return c.timestamp
}

// SetTimestamp implements ChangeView interface.
func (c *changeView) SetTimestamp(ts uint32) {
	c.timestamp = ts
}
