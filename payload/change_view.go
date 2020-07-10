package payload

import "github.com/nspcc-dev/neo-go/pkg/io"

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
}

type changeView struct {
	newViewNumber byte
	timestamp     uint32
}

var _ ChangeView = (*changeView)(nil)

// EncodeBinary implements io.Serializable interface.
func (c changeView) EncodeBinary(w *io.BinWriter) {
	w.WriteU32LE(c.timestamp)
}

// DecodeBinary implements io.Serializable interface.
func (c *changeView) DecodeBinary(r *io.BinReader) {
	c.timestamp = r.ReadU32LE()
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
	return uint64(c.timestamp) * 1000000000
}

// SetTimestamp implements ChangeView interface.
func (c *changeView) SetTimestamp(ts uint64) {
	c.timestamp = uint32(ts / 1000000000)
}
