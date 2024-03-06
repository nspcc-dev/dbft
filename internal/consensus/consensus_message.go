package consensus

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/nspcc-dev/dbft"
	"github.com/nspcc-dev/dbft/internal/crypto"
)

type (
	// Serializable is an interface for serializing consensus messages.
	Serializable interface {
		EncodeBinary(encoder *gob.Encoder) error
		DecodeBinary(decoder *gob.Decoder) error
	}

	message struct {
		cmType     dbft.MessageType
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

var _ dbft.ConsensusMessage[crypto.Uint256] = (*message)(nil)

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
	m.cmType = dbft.MessageType(aux.CMType)
	m.viewNumber = aux.ViewNumber

	switch m.cmType {
	case dbft.ChangeViewType:
		cv := new(changeView)
		cv.newViewNumber = m.viewNumber + 1
		m.payload = cv
	case dbft.PrepareRequestType:
		m.payload = new(prepareRequest)
	case dbft.PrepareResponseType:
		m.payload = new(prepareResponse)
	case dbft.CommitType:
		m.payload = new(commit)
	case dbft.RecoveryRequestType:
		m.payload = new(recoveryRequest)
	case dbft.RecoveryMessageType:
		m.payload = new(recoveryMessage)
	default:
		return fmt.Errorf("invalid type: 0x%02x", byte(m.cmType))
	}

	rr := bytes.NewReader(aux.Payload)
	dec := gob.NewDecoder(rr)
	return m.payload.(Serializable).DecodeBinary(dec)
}

func (m message) GetChangeView() dbft.ChangeView { return m.payload.(dbft.ChangeView) }
func (m message) GetPrepareRequest() dbft.PrepareRequest[crypto.Uint256] {
	return m.payload.(dbft.PrepareRequest[crypto.Uint256])
}
func (m message) GetPrepareResponse() dbft.PrepareResponse[crypto.Uint256] {
	return m.payload.(dbft.PrepareResponse[crypto.Uint256])
}
func (m message) GetCommit() dbft.Commit                   { return m.payload.(dbft.Commit) }
func (m message) GetRecoveryRequest() dbft.RecoveryRequest { return m.payload.(dbft.RecoveryRequest) }
func (m message) GetRecoveryMessage() dbft.RecoveryMessage[crypto.Uint256] {
	return m.payload.(dbft.RecoveryMessage[crypto.Uint256])
}

// ViewNumber implements ConsensusMessage interface.
func (m message) ViewNumber() byte {
	return m.viewNumber
}

// Type implements ConsensusMessage interface.
func (m message) Type() dbft.MessageType {
	return m.cmType
}

// Payload implements ConsensusMessage interface.
func (m message) Payload() any {
	return m.payload
}
