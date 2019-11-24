package payload

import (
	"errors"

	"github.com/CityOfZion/neo-go/pkg/io"
	"github.com/CityOfZion/neo-go/pkg/util"
)

type (
	// RecoveryMessage represents dBFT Recovery message.
	RecoveryMessage interface {
		// AddPayload adds payload from this epoch to be recovered.
		AddPayload(p ConsensusPayload)
		// GetPrepareRequest returns PrepareRequest to be processed.
		GetPrepareRequest(p ConsensusPayload) ConsensusPayload
		// GetPrepareResponses returns a slice of PrepareResponse in any order.
		GetPrepareResponses(p ConsensusPayload) []ConsensusPayload
		// GetChangeView returns a slice of ChangeView in any order.
		GetChangeViews(p ConsensusPayload) []ConsensusPayload
		// GetCommits returns a slice of Commit in any order.
		GetCommits(p ConsensusPayload) []ConsensusPayload

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
	case PrepareResponseType:
		m.preparationPayloads = append(m.preparationPayloads, preparationCompact{
			validatorIndex: p.ValidatorIndex(),
		})
	case ChangeViewType:
		m.changeViewPayloads = append(m.changeViewPayloads, changeViewCompact{
			validatorIndex:     p.ValidatorIndex(),
			originalViewNumber: p.ViewNumber(),
			timestamp:          0,
		})
	case CommitType:
		cc := commitCompact{
			viewNumber:     p.ViewNumber(),
			validatorIndex: p.ValidatorIndex(),
		}
		copy(cc.signature[:], p.GetCommit().Signature())
		m.commitPayloads = append(m.commitPayloads, cc)
	}
}

func fromPayload(t MessageType, recovery ConsensusPayload, p interface{}) *consensusPayload {
	return &consensusPayload{
		message: message{
			cmType:     t,
			viewNumber: recovery.ViewNumber(),
			payload:    p,
		},
		version:  recovery.Version(),
		prevHash: recovery.PrevHash(),
		height:   recovery.Height(),
	}
}

// GetPrepareRequest implements RecoveryMessage interface.
func (m *recoveryMessage) GetPrepareRequest(p ConsensusPayload) ConsensusPayload {
	return fromPayload(PrepareRequestType, p, &prepareRequest{
		timestamp:         m.prepareRequest.Timestamp(),
		nonce:             m.prepareRequest.Nonce(),
		transactionHashes: m.prepareRequest.TransactionHashes(),
		nextConsensus:     m.prepareRequest.NextConsensus(),
	})
}

// GetPrepareResponses implements RecoveryMessage interface.
func (m *recoveryMessage) GetPrepareResponses(p ConsensusPayload) []ConsensusPayload {
	if m.preparationHash == nil {
		return nil
	}

	payloads := make([]ConsensusPayload, len(m.preparationPayloads))

	for i, resp := range m.preparationPayloads {
		payloads[i] = fromPayload(PrepareResponseType, p, &prepareResponse{
			preparationHash: *m.preparationHash,
		})
		payloads[i].SetValidatorIndex(resp.validatorIndex)
	}

	return payloads
}

// GetChangeViews implements RecoveryMessage interface.
func (m *recoveryMessage) GetChangeViews(p ConsensusPayload) []ConsensusPayload {
	payloads := make([]ConsensusPayload, len(m.changeViewPayloads))

	for i, cv := range m.changeViewPayloads {
		payloads[i] = fromPayload(ChangeViewType, p, &changeView{
			newViewNumber: cv.originalViewNumber + 1,
			timestamp:     cv.timestamp,
		})
		payloads[i].SetValidatorIndex(cv.validatorIndex)
	}

	return payloads
}

// GetCommits implements RecoveryMessage interface.
func (m *recoveryMessage) GetCommits(p ConsensusPayload) []ConsensusPayload {
	payloads := make([]ConsensusPayload, len(m.commitPayloads))

	for i, c := range m.commitPayloads {
		payloads[i] = fromPayload(CommitType, p, &commit{signature: c.signature})
		payloads[i].SetValidatorIndex(c.validatorIndex)
	}

	return payloads
}

// EncodeBinary implements io.Serializable interface.
func (m recoveryMessage) EncodeBinary(w *io.BinWriter) {
	w.WriteArray(m.changeViewPayloads)

	hasReq := m.prepareRequest != nil
	w.WriteLE(hasReq)

	if hasReq {
		m.prepareRequest.(io.Serializable).EncodeBinary(w)
	} else {
		if m.preparationHash == nil {
			w.WriteVarUint(0)
		} else {
			w.WriteVarUint(util.Uint256Size)
			m.preparationHash.EncodeBinary(w)
		}
	}

	w.WriteArray(m.preparationPayloads)
	w.WriteArray(m.commitPayloads)
}

// DecodeBinary implements io.Serializable interface.
func (m *recoveryMessage) DecodeBinary(r *io.BinReader) {
	r.ReadArray(&m.changeViewPayloads)

	var hasReq bool

	r.ReadLE(&hasReq)

	if hasReq {
		m.prepareRequest = new(prepareRequest)
		m.prepareRequest.(io.Serializable).DecodeBinary(r)
	} else {
		l := r.ReadVarUint()
		if l != 0 {
			if l == util.Uint256Size {
				m.preparationHash = new(util.Uint256)
				m.preparationHash.DecodeBinary(r)
			} else {
				r.Err = errors.New("wrong util.Uint256 length")
			}
		} else {
			m.preparationHash = nil
		}
	}

	r.ReadArray(&m.preparationPayloads)
	r.ReadArray(&m.commitPayloads)
}
