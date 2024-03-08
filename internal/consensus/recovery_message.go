package consensus

import (
	"encoding/gob"
	"errors"

	"github.com/nspcc-dev/dbft"
	"github.com/nspcc-dev/dbft/internal/crypto"
)

type (
	recoveryMessage struct {
		preparationHash     *crypto.Uint256
		preparationPayloads []preparationCompact
		commitPayloads      []commitCompact
		changeViewPayloads  []changeViewCompact
		prepareRequest      dbft.PrepareRequest[crypto.Uint256]
	}
	// recoveryMessageAux is an auxiliary structure for recoveryMessage encoding.
	recoveryMessageAux struct {
		PreparationPayloads []preparationCompact
		CommitPayloads      []commitCompact
		ChangeViewPayloads  []changeViewCompact
	}
)

var _ dbft.RecoveryMessage[crypto.Uint256] = (*recoveryMessage)(nil)

// PreparationHash implements RecoveryMessage interface.
func (m *recoveryMessage) PreparationHash() *crypto.Uint256 {
	return m.preparationHash
}

// AddPayload implements RecoveryMessage interface.
func (m *recoveryMessage) AddPayload(p dbft.ConsensusPayload[crypto.Uint256]) {
	switch p.Type() {
	case dbft.PrepareRequestType:
		m.prepareRequest = p.GetPrepareRequest()
		prepHash := p.Hash()
		m.preparationHash = &prepHash
	case dbft.PrepareResponseType:
		m.preparationPayloads = append(m.preparationPayloads, preparationCompact{
			ValidatorIndex: p.ValidatorIndex(),
		})
	case dbft.ChangeViewType:
		m.changeViewPayloads = append(m.changeViewPayloads, changeViewCompact{
			ValidatorIndex:     p.ValidatorIndex(),
			OriginalViewNumber: p.ViewNumber(),
			Timestamp:          0,
		})
	case dbft.CommitType:
		cc := commitCompact{
			ViewNumber:     p.ViewNumber(),
			ValidatorIndex: p.ValidatorIndex(),
		}
		copy(cc.Signature[:], p.GetCommit().Signature())
		m.commitPayloads = append(m.commitPayloads, cc)
	}
}

func fromPayload(t dbft.MessageType, recovery dbft.ConsensusPayload[crypto.Uint256], p Serializable) *Payload {
	return &Payload{
		message: message{
			cmType:     t,
			viewNumber: recovery.ViewNumber(),
			payload:    p,
		},
		height: recovery.Height(),
	}
}

// GetPrepareRequest implements RecoveryMessage interface.
func (m *recoveryMessage) GetPrepareRequest(p dbft.ConsensusPayload[crypto.Uint256], _ []dbft.PublicKey, ind uint16) dbft.ConsensusPayload[crypto.Uint256] {
	if m.prepareRequest == nil {
		return nil
	}

	req := fromPayload(dbft.PrepareRequestType, p, &prepareRequest{
		// prepareRequest.Timestamp() here returns nanoseconds-precision value, so convert it to seconds again
		timestamp:         nanoSecToSec(m.prepareRequest.Timestamp()),
		nonce:             m.prepareRequest.Nonce(),
		transactionHashes: m.prepareRequest.TransactionHashes(),
	})
	req.SetValidatorIndex(ind)

	return req
}

// GetPrepareResponses implements RecoveryMessage interface.
func (m *recoveryMessage) GetPrepareResponses(p dbft.ConsensusPayload[crypto.Uint256], _ []dbft.PublicKey) []dbft.ConsensusPayload[crypto.Uint256] {
	if m.preparationHash == nil {
		return nil
	}

	payloads := make([]dbft.ConsensusPayload[crypto.Uint256], len(m.preparationPayloads))

	for i, resp := range m.preparationPayloads {
		payloads[i] = fromPayload(dbft.PrepareResponseType, p, &prepareResponse{
			preparationHash: *m.preparationHash,
		})
		payloads[i].SetValidatorIndex(resp.ValidatorIndex)
	}

	return payloads
}

// GetChangeViews implements RecoveryMessage interface.
func (m *recoveryMessage) GetChangeViews(p dbft.ConsensusPayload[crypto.Uint256], _ []dbft.PublicKey) []dbft.ConsensusPayload[crypto.Uint256] {
	payloads := make([]dbft.ConsensusPayload[crypto.Uint256], len(m.changeViewPayloads))

	for i, cv := range m.changeViewPayloads {
		payloads[i] = fromPayload(dbft.ChangeViewType, p, &changeView{
			newViewNumber: cv.OriginalViewNumber + 1,
			timestamp:     cv.Timestamp,
		})
		payloads[i].SetValidatorIndex(cv.ValidatorIndex)
	}

	return payloads
}

// GetCommits implements RecoveryMessage interface.
func (m *recoveryMessage) GetCommits(p dbft.ConsensusPayload[crypto.Uint256], _ []dbft.PublicKey) []dbft.ConsensusPayload[crypto.Uint256] {
	payloads := make([]dbft.ConsensusPayload[crypto.Uint256], len(m.commitPayloads))

	for i, c := range m.commitPayloads {
		payloads[i] = fromPayload(dbft.CommitType, p, &commit{signature: c.Signature})
		payloads[i].SetValidatorIndex(c.ValidatorIndex)
	}

	return payloads
}

// EncodeBinary implements Serializable interface.
func (m recoveryMessage) EncodeBinary(w *gob.Encoder) error {
	hasReq := m.prepareRequest != nil
	if err := w.Encode(hasReq); err != nil {
		return err
	}
	if hasReq {
		if err := m.prepareRequest.(Serializable).EncodeBinary(w); err != nil {
			return err
		}
	} else {
		if m.preparationHash == nil {
			if err := w.Encode(0); err != nil {
				return err
			}
		} else {
			if err := w.Encode(crypto.Uint256Size); err != nil {
				return err
			}
			if err := w.Encode(m.preparationHash); err != nil {
				return err
			}
		}
	}
	return w.Encode(&recoveryMessageAux{
		PreparationPayloads: m.preparationPayloads,
		CommitPayloads:      m.commitPayloads,
		ChangeViewPayloads:  m.changeViewPayloads,
	})
}

// DecodeBinary implements Serializable interface.
func (m *recoveryMessage) DecodeBinary(r *gob.Decoder) error {
	var hasReq bool
	if err := r.Decode(&hasReq); err != nil {
		return err
	}
	if hasReq {
		m.prepareRequest = new(prepareRequest)
		if err := m.prepareRequest.(Serializable).DecodeBinary(r); err != nil {
			return err
		}
	} else {
		var l int
		if err := r.Decode(&l); err != nil {
			return err
		}
		if l != 0 {
			if l == crypto.Uint256Size {
				m.preparationHash = new(crypto.Uint256)
				if err := r.Decode(m.preparationHash); err != nil {
					return err
				}
			} else {
				return errors.New("wrong crypto.Uint256 length")
			}
		} else {
			m.preparationHash = nil
		}
	}

	aux := new(recoveryMessageAux)
	if err := r.Decode(aux); err != nil {
		return err
	}
	m.preparationPayloads = aux.PreparationPayloads
	if m.preparationPayloads == nil {
		m.preparationPayloads = []preparationCompact{}
	}
	m.commitPayloads = aux.CommitPayloads
	if m.commitPayloads == nil {
		m.commitPayloads = []commitCompact{}
	}
	m.changeViewPayloads = aux.ChangeViewPayloads
	if m.changeViewPayloads == nil {
		m.changeViewPayloads = []changeViewCompact{}
	}
	return nil
}
