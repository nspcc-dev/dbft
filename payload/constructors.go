package payload

import (
	"github.com/nspcc-dev/dbft"
	"github.com/nspcc-dev/dbft/crypto"
)

// NewConsensusPayload returns minimal ConsensusPayload implementation.
func NewConsensusPayload() dbft.ConsensusPayload[crypto.Uint256, crypto.Uint160] {
	return &Payload{}
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
func NewChangeView() dbft.ChangeView {
	return new(changeView)
}

// NewCommit returns minimal Commit implementation.
func NewCommit() dbft.Commit {
	return new(commit)
}

// NewRecoveryRequest returns minimal RecoveryRequest implementation.
func NewRecoveryRequest() dbft.RecoveryRequest {
	return new(recoveryRequest)
}

// NewRecoveryMessage returns minimal RecoveryMessage implementation.
func NewRecoveryMessage() dbft.RecoveryMessage[crypto.Uint256, crypto.Uint160] {
	return &recoveryMessage{
		preparationPayloads: make([]preparationCompact, 0),
		commitPayloads:      make([]commitCompact, 0),
		changeViewPayloads:  make([]changeViewCompact, 0),
	}
}
