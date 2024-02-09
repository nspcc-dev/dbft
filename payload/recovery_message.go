package payload

import (
	"encoding/gob"
	"errors"

	"github.com/nspcc-dev/dbft/crypto"
	"github.com/nspcc-dev/neo-go/pkg/util"
)

type (
	// RecoveryMessage represents dBFT Recovery message.
	RecoveryMessage interface {
		// AddPayload adds payload from this epoch to be recovered.
		AddPayload(p ConsensusPayload)
		// GetPrepareRequest returns PrepareRequest to be processed.
		GetPrepareRequest(p ConsensusPayload, validators []crypto.PublicKey, primary uint16) ConsensusPayload
		// GetPrepareResponses returns a slice of PrepareResponse in any order.
		GetPrepareResponses(p ConsensusPayload, validators []crypto.PublicKey) []ConsensusPayload
		// GetChangeViews returns a slice of ChangeView in any order.
		GetChangeViews(p ConsensusPayload, validators []crypto.PublicKey) []ConsensusPayload
		// GetCommits returns a slice of Commit in any order.
		GetCommits(p ConsensusPayload, validators []crypto.PublicKey) []ConsensusPayload

		// PreparationHash returns has of PrepareRequest payload for this epoch.
		// It can be useful in case only PrepareResponse payloads were received.
		PreparationHash() *util.Uint256
		// SetPreparationHash sets preparation hash.
		SetPreparationHash(h *util.Uint256)
	}

	recoveryMessage struct {
		preparationHash     *util.Uint256
		preparationPayloads []preparationCompact
		commitPayloads      []commitCompact
		changeViewPayloads  []changeViewCompact
		prepareRequest      PrepareRequest
	}
	// recoveryMessageAux is an auxiliary structure for recoveryMessage encoding.
	recoveryMessageAux struct {
		PreparationPayloads []preparationCompact
		CommitPayloads      []commitCompact
		ChangeViewPayloads  []changeViewCompact
	}
)

var _ RecoveryMessage = (*recoveryMessage)(nil)

// PreparationHash implements RecoveryMessage interface.
func (m *recoveryMessage) PreparationHash() *util.Uint256 {
	return m.preparationHash
}

// SetPreparationHash implements RecoveryMessage interface.
func (m *recoveryMessage) SetPreparationHash(h *util.Uint256) {
	m.preparationHash = h
}

// AddPayload implements RecoveryMessage interface.
func (m *recoveryMessage) AddPayload(p ConsensusPayload) {
	switch p.Type() {
	case PrepareRequestType:
		m.prepareRequest = p.GetPrepareRequest()
		prepHash := p.Hash()
		m.preparationHash = &prepHash
	case PrepareResponseType:
		m.preparationPayloads = append(m.preparationPayloads, preparationCompact{
			ValidatorIndex: p.ValidatorIndex(),
		})
	case ChangeViewType:
		m.changeViewPayloads = append(m.changeViewPayloads, changeViewCompact{
			ValidatorIndex:     p.ValidatorIndex(),
			OriginalViewNumber: p.ViewNumber(),
			Timestamp:          0,
		})
	case CommitType:
		cc := commitCompact{
			ViewNumber:     p.ViewNumber(),
			ValidatorIndex: p.ValidatorIndex(),
		}
		copy(cc.Signature[:], p.GetCommit().Signature())
		m.commitPayloads = append(m.commitPayloads, cc)
	}
}

func fromPayload(t MessageType, recovery ConsensusPayload, p Serializable) *Payload {
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
func (m *recoveryMessage) GetPrepareRequest(p ConsensusPayload, _ []crypto.PublicKey, ind uint16) ConsensusPayload {
	if m.prepareRequest == nil {
		return nil
	}

	req := fromPayload(PrepareRequestType, p, &prepareRequest{
		// prepareRequest.Timestamp() here returns nanoseconds-precision value, so convert it to seconds again
		timestamp:         nanoSecToSec(m.prepareRequest.Timestamp()),
		nonce:             m.prepareRequest.Nonce(),
		transactionHashes: m.prepareRequest.TransactionHashes(),
		nextConsensus:     m.prepareRequest.NextConsensus(),
	})
	req.SetValidatorIndex(ind)

	return req
}

// GetPrepareResponses implements RecoveryMessage interface.
func (m *recoveryMessage) GetPrepareResponses(p ConsensusPayload, _ []crypto.PublicKey) []ConsensusPayload {
	if m.preparationHash == nil {
		return nil
	}

	payloads := make([]ConsensusPayload, len(m.preparationPayloads))

	for i, resp := range m.preparationPayloads {
		payloads[i] = fromPayload(PrepareResponseType, p, &prepareResponse{
			preparationHash: *m.preparationHash,
		})
		payloads[i].SetValidatorIndex(resp.ValidatorIndex)
	}

	return payloads
}

// GetChangeViews implements RecoveryMessage interface.
func (m *recoveryMessage) GetChangeViews(p ConsensusPayload, _ []crypto.PublicKey) []ConsensusPayload {
	payloads := make([]ConsensusPayload, len(m.changeViewPayloads))

	for i, cv := range m.changeViewPayloads {
		payloads[i] = fromPayload(ChangeViewType, p, &changeView{
			newViewNumber: cv.OriginalViewNumber + 1,
			timestamp:     cv.Timestamp,
		})
		payloads[i].SetValidatorIndex(cv.ValidatorIndex)
	}

	return payloads
}

// GetCommits implements RecoveryMessage interface.
func (m *recoveryMessage) GetCommits(p ConsensusPayload, _ []crypto.PublicKey) []ConsensusPayload {
	payloads := make([]ConsensusPayload, len(m.commitPayloads))

	for i, c := range m.commitPayloads {
		payloads[i] = fromPayload(CommitType, p, &commit{signature: c.Signature})
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
			if err := w.Encode(util.Uint256Size); err != nil {
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
			if l == util.Uint256Size {
				m.preparationHash = new(util.Uint256)
				if err := r.Decode(m.preparationHash); err != nil {
					return err
				}
			} else {
				return errors.New("wrong util.Uint256 length")
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
