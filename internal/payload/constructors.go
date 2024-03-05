package payload

import (
	"github.com/nspcc-dev/dbft"
	"github.com/nspcc-dev/dbft/internal/crypto"
)

// NewConsensusPayload returns minimal ConsensusPayload implementation.
func NewConsensusPayload(t dbft.MessageType, height uint32, validatorIndex uint16, viewNumber byte, consensusMessage any) dbft.ConsensusPayload[crypto.Uint256, crypto.Uint160] {
	return &Payload{
		message: message{
			cmType:     t,
			viewNumber: viewNumber,
			payload:    consensusMessage,
		},
		validatorIndex: validatorIndex,
		height:         height,
	}
}

// NewPrepareRequest returns minimal prepareRequest implementation.
func NewPrepareRequest() dbft.PrepareRequest[crypto.Uint256, crypto.Uint160] {
	return new(prepareRequest)
}

// NewPrepareResponse returns minimal PrepareResponse implementation.
func NewPrepareResponse() dbft.PrepareResponse[crypto.Uint256] {
	return new(prepareResponse)
}

// NewChangeView returns minimal ChangeView implementation.
func NewChangeView(newViewNumber byte, _ dbft.ChangeViewReason, ts uint64) dbft.ChangeView {
	return &changeView{
		newViewNumber: newViewNumber,
		timestamp:     nanoSecToSec(ts),
	}
}

// NewCommit returns minimal Commit implementation.
func NewCommit(signature []byte) dbft.Commit {
	c := new(commit)
	copy(c.signature[:], signature)
	return c
}

// NewRecoveryRequest returns minimal RecoveryRequest implementation.
func NewRecoveryRequest() dbft.RecoveryRequest {
	return new(recoveryRequest)
}

// NewRecoveryMessage returns minimal RecoveryMessage implementation.
func NewRecoveryMessage(preparationHash *crypto.Uint256) dbft.RecoveryMessage[crypto.Uint256, crypto.Uint160] {
	return &recoveryMessage{
		preparationHash:     preparationHash,
		preparationPayloads: make([]preparationCompact, 0),
		commitPayloads:      make([]commitCompact, 0),
		changeViewPayloads:  make([]changeViewCompact, 0),
	}
}
