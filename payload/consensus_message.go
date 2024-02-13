package payload

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/nspcc-dev/dbft/crypto"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/pkg/errors"
)

type (
	// MessageType is a type for dBFT consensus messages.
	MessageType byte

	// Serializable is an interface for serializing consensus messages.
	Serializable interface {
		EncodeBinary(encoder *gob.Encoder) error
		DecodeBinary(decoder *gob.Decoder) error
	}

	consensusMessage[H crypto.Hash, A crypto.Address] interface {
		// ViewNumber returns view number when this message was originated.
		ViewNumber() byte
		// SetViewNumber sets view number.
		SetViewNumber(view byte)

		// Type returns type of this message.
		Type() MessageType
		// SetType sets the type of this message.
		SetType(t MessageType)

		// Payload returns this message's actual payload.
		Payload() any
		// SetPayload sets this message's payload to p.
		SetPayload(p any)

		// GetChangeView returns payload as if it was ChangeView.
		GetChangeView() ChangeView
		// GetPrepareRequest returns payload as if it was PrepareRequest.
		GetPrepareRequest() PrepareRequest[H, A]
		// GetPrepareResponse returns payload as if it was PrepareResponse.
		GetPrepareResponse() PrepareResponse[H]
		// GetCommit returns payload as if it was Commit.
		GetCommit() Commit
		// GetRecoveryRequest returns payload as if it was RecoveryRequest.
		GetRecoveryRequest() RecoveryRequest
		// GetRecoveryMessage returns payload as if it was RecoveryMessage.
		GetRecoveryMessage() RecoveryMessage[H, A]
	}

	message struct {
		cmType     MessageType
		viewNumber byte

		payload any
	}

	// messageAux is an auxiliary structure for message marshalling.
	messageAux struct {
		CMType     byte
		ViewNumber byte
		Payload    []byte
	}
)

// 6 following constants enumerate all possible type of consensus message.
const (
	ChangeViewType      MessageType = 0x00
	PrepareRequestType  MessageType = 0x20
	PrepareResponseType MessageType = 0x21
	CommitType          MessageType = 0x30
	RecoveryRequestType MessageType = 0x40
	RecoveryMessageType MessageType = 0x41
)

var _ consensusMessage[util.Uint256, util.Uint160] = (*message)(nil)

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
		return fmt.Sprintf("UNKNOWN(%02x)", byte(m))
	}
}

// EncodeBinary implements Serializable interface.
func (m message) EncodeBinary(w *gob.Encoder) error {
	ww := bytes.Buffer{}
	enc := gob.NewEncoder(&ww)
	if err := m.payload.(Serializable).EncodeBinary(enc); err != nil {
		return err
	}
	return w.Encode(&messageAux{
		CMType:     byte(m.cmType),
		ViewNumber: m.viewNumber,
		Payload:    ww.Bytes(),
	})
}

// DecodeBinary implements Serializable interface.
func (m *message) DecodeBinary(r *gob.Decoder) error {
	aux := new(messageAux)
	if err := r.Decode(aux); err != nil {
		return err
	}
	m.cmType = MessageType(aux.CMType)
	m.viewNumber = aux.ViewNumber

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
		return errors.Errorf("invalid type: 0x%02x", byte(m.cmType))
	}

	rr := bytes.NewReader(aux.Payload)
	dec := gob.NewDecoder(rr)
	return m.payload.(Serializable).DecodeBinary(dec)
}

func (m message) GetChangeView() ChangeView { return m.payload.(ChangeView) }
func (m message) GetPrepareRequest() PrepareRequest[util.Uint256, util.Uint160] {
	return m.payload.(PrepareRequest[util.Uint256, util.Uint160])
}
func (m message) GetPrepareResponse() PrepareResponse[util.Uint256] {
	return m.payload.(PrepareResponse[util.Uint256])
}
func (m message) GetCommit() Commit                   { return m.payload.(Commit) }
func (m message) GetRecoveryRequest() RecoveryRequest { return m.payload.(RecoveryRequest) }
func (m message) GetRecoveryMessage() RecoveryMessage[util.Uint256, util.Uint160] {
	return m.payload.(RecoveryMessage[util.Uint256, util.Uint160])
}

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
func (m message) Payload() any {
	return m.payload
}

// SetPayload implements ConsensusMessage interface.
func (m *message) SetPayload(p any) {
	m.payload = p
}
