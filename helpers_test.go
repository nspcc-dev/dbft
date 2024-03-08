package dbft

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Structures used for type-specific dBFT/payloads implementation to avoid cyclic
// dependency.
type (
	hash        struct{}
	payloadStub struct {
		height         uint32
		typ            MessageType
		validatorIndex uint16
	}
)

func (hash) String() string {
	return ""
}

func (p payloadStub) ViewNumber() byte {
	panic("TODO")
}
func (p payloadStub) SetViewNumber(byte) {
	panic("TODO")
}
func (p payloadStub) Type() MessageType {
	return p.typ
}
func (p payloadStub) SetType(MessageType) {
	panic("TODO")
}
func (p payloadStub) Payload() any {
	panic("TODO")
}
func (p payloadStub) SetPayload(any) {
	panic("TODO")
}
func (p payloadStub) GetChangeView() ChangeView {
	panic("TODO")
}
func (p payloadStub) GetPrepareRequest() PrepareRequest[hash] {
	panic("TODO")
}
func (p payloadStub) GetPrepareResponse() PrepareResponse[hash] {
	panic("TODO")
}
func (p payloadStub) GetCommit() Commit {
	panic("TODO")
}
func (p payloadStub) GetRecoveryRequest() RecoveryRequest {
	panic("TODO")
}
func (p payloadStub) GetRecoveryMessage() RecoveryMessage[hash] {
	panic("TODO")
}
func (p payloadStub) ValidatorIndex() uint16 {
	return p.validatorIndex
}
func (p payloadStub) SetValidatorIndex(uint16) {
	panic("TODO")
}
func (p payloadStub) Height() uint32 {
	return p.height
}
func (p payloadStub) SetHeight(uint32) {
	panic("TODO")
}
func (p payloadStub) Hash() hash {
	panic("TODO")
}

func TestMessageCache(t *testing.T) {
	c := newCache[hash]()

	p1 := payloadStub{
		height: 3,
		typ:    PrepareRequestType,
	}
	c.addMessage(p1)

	p2 := payloadStub{
		height: 4,
		typ:    ChangeViewType,
	}
	c.addMessage(p2)

	p3 := payloadStub{
		height: 4,
		typ:    CommitType,
	}
	c.addMessage(p3)

	box := c.getHeight(3)
	require.Len(t, box.chViews, 0)
	require.Len(t, box.prepare, 1)
	require.Len(t, box.commit, 0)

	box = c.getHeight(4)
	require.Len(t, box.chViews, 1)
	require.Len(t, box.prepare, 0)
	require.Len(t, box.commit, 1)
}
