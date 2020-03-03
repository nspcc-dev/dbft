package payload

import "github.com/nspcc-dev/neo-go/pkg/io"

// Commit is an interface for dBFT Commit message.
type Commit interface {
	// Signature returns commit's signature field
	// which is a block signature for the current epoch.
	Signature() []byte

	// SetSignature sets commit's signature.
	SetSignature(signature []byte)
}

type commit struct {
	signature [signatureSize]byte
}

const signatureSize = 64

var _ Commit = (*commit)(nil)

// EncodeBinary implements io.Serializable interface.
func (c commit) EncodeBinary(w *io.BinWriter) {
	w.WriteBytes(c.signature[:])
}

// DecodeBinary implements io.Serializable interface.
func (c *commit) DecodeBinary(r *io.BinReader) {
	r.ReadBytes(c.signature[:])
}

// Signature implements Commit interface.
func (c commit) Signature() []byte {
	return c.signature[:]
}

// SetSignature implements Commit interface.
func (c *commit) SetSignature(sig []byte) {
	copy(c.signature[:], sig)
}
