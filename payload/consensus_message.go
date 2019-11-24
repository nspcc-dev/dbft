package payload

import (
	"github.com/CityOfZion/neo-go/pkg/io"
	"github.com/pkg/errors"
)

type (
	MessageType byte

	consensusMessage interface {
		// ViewNumber returns view number when this message was originated.
		ViewNumber() byte
		// SetViewNumber sets view number.
		SetViewNumber(view byte)

		// Type returns type of this message.
		Type() MessageType
		// SetType sets the type of this message.
		SetType(t MessageType)

		// Payload returns this message's actual payload.
		Payload() interface{}
		// SetPayload sets this message's payload to p.
		SetPayload(p interface{})

		// GetChangeView returns payload as if it was ChangeView.
		GetChangeView() ChangeView
		// GetPrepareRequest returns payload as if it was PrepareRequest.
		GetPrepareRequest() PrepareRequest
		// GetPrepareResponse returns payload as if it was PrepareResponse.
		GetPrepareResponse() PrepareResponse
		// GetCommit returns payload as if it was Commit.
		GetCommit() Commit
		// GetRecoveryRequest returns payload as if it was RecoveryRequest.
		GetRecoveryRequest() RecoveryRequest
		// GetRecoveryMessage returns payload as if it was RecoveryMessage.
		GetRecoveryMessage() RecoveryMessage
	}

	message struct {
		cmType     MessageType
		viewNumber byte

		payload interface{}
	}
)

const (
	ChangeViewType      MessageType = 0x00
	PrepareRequestType  MessageType = 0x20
	PrepareResponseType MessageType = 0x21
	CommitType          MessageType = 0x30
	RecoveryRequestType MessageType = 0x40
	RecoveryMessageType MessageType = 0x41
)

var _ consensusMessage = (*message)(nil)

// String implements fmt.Stringer interface.
func (m MessageType) String() string {
	switch m {
	case ChangeViewType:
		return "ChangeView"
	case PrepareRequestType:
		return "PrepareRequest"
	case PrepareResponseType:
		return "PrepareResponse"
	case CommitType:
		return "Commit"
	case RecoveryRequestType:
		return "RecoveryRequest"
	case RecoveryMessageType:
		return "RecoveryMessage"
	default:
		panic("unknown type")
	}
}

// EncodeBinary implements io.Serializable interface.
func (m message) EncodeBinary(w *io.BinWriter) {
	w.WriteLE(byte(m.cmType))
	w.WriteLE(m.viewNumber)
	m.payload.(io.Serializable).EncodeBinary(w)
}

// DecodeBinary implements io.Serializable interface.
func (m *message) DecodeBinary(r *io.BinReader) {
	r.ReadLE((*byte)(&m.cmType))
	r.ReadLE(&m.viewNumber)

	switch m.cmType {
	case ChangeViewType:
		cv := new(changeView)
		cv.newViewNumber = m.viewNumber + 1
		m.payload = cv
	case PrepareRequestType:
		m.payload = new(prepareRequest)
	case PrepareResponseType:
		m.payload = new(prepareResponse)
	case CommitType:
		m.payload = new(commit)
	case RecoveryRequestType:
		m.payload = new(recoveryRequest)
	case RecoveryMessageType:
		m.payload = new(recoveryMessage)
	default:
		r.Err = errors.Errorf("invalid type: 0x%02x", byte(m.cmType))
		return
	}

	m.payload.(io.Serializable).DecodeBinary(r)
}

func (m message) GetChangeView() ChangeView           { return m.payload.(ChangeView) }
func (m message) GetPrepareRequest() PrepareRequest   { return m.payload.(PrepareRequest) }
func (m message) GetPrepareResponse() PrepareResponse { return m.payload.(PrepareResponse) }
func (m message) GetCommit() Commit                   { return m.payload.(Commit) }
func (m message) GetRecoveryRequest() RecoveryRequest { return m.payload.(RecoveryRequest) }
func (m message) GetRecoveryMessage() RecoveryMessage { return m.payload.(RecoveryMessage) }

// ViewNumber implements ConsensusMessage interface.
func (m message) ViewNumber() byte {
	return m.viewNumber
}

// SetViewNumber implements ConsensusMessage interface.
func (m *message) SetViewNumber(view byte) {
	m.viewNumber = view
}

// Type implements ConsensusMessage interface.
func (m message) Type() MessageType {
	return m.cmType
}

// SetType implements ConsensusMessage interface.
func (m *message) SetType(t MessageType) {
	m.cmType = t
}

// Payload implements ConsensusMessage interface.
func (m message) Payload() interface{} {
	return m.payload
}

// SetPayload implements ConsensusMessage interface.
func (m *message) SetPayload(p interface{}) {
	m.payload = p
}
