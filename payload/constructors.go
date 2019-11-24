package payload

// NewConsensusPayload returns minimal ConsensusPayload implementation.
func NewConsensusPayload() ConsensusPayload {
	return &consensusPayload{}
}

// NewPrepareRequest returns minimal prepareRequest implementation.
func NewPrepareRequest() PrepareRequest {
	return new(prepareRequest)
}

// NewPrepareResponse returns minimal PrepareResponse implementation.
func NewPrepareResponse() PrepareResponse {
	return new(prepareResponse)
}

// NewChangeView returns minimal ChangeView implementation.
func NewChangeView() ChangeView {
	return new(changeView)
}

// NewCommit returns minimal Commit implementation.
func NewCommit() Commit {
	return new(commit)
}

// NewRecoveryRequest returns minimal RecoveryRequest implementation.
func NewRecoveryRequest() RecoveryRequest {
	return new(recoveryRequest)
}

// NewRecoveryMessage returns minimal RecoveryMessage implementation.
func NewRecoveryMessage() RecoveryMessage {
	return &recoveryMessage{
		preparationPayloads: make([]preparationCompact, 0),
		commitPayloads:      make([]commitCompact, 0),
		changeViewPayloads:  make([]changeViewCompact, 0),
	}
}
