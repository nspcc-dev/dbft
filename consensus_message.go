package dbft

// ConsensusMessage is an interface for generic dBFT message.
type ConsensusMessage[H Hash] interface {
	// ViewNumber returns view number when this message was originated.
	ViewNumber() byte
	// Type returns type of this message.
	Type() MessageType
	// Payload returns this message's actual payload.
	Payload() any

	// GetChangeView returns payload as if it was ChangeView.
	GetChangeView() ChangeView
	// GetPrepareRequest returns payload as if it was PrepareRequest.
	GetPrepareRequest() PrepareRequest[H]
	// GetPrepareResponse returns payload as if it was PrepareResponse.
	GetPrepareResponse() PrepareResponse[H]
	// GetCommit returns payload as if it was Commit.
	GetCommit() Commit
	// GetRecoveryRequest returns payload as if it was RecoveryRequest.
	GetRecoveryRequest() RecoveryRequest
	// GetRecoveryMessage returns payload as if it was RecoveryMessage.
	GetRecoveryMessage() RecoveryMessage[H]
}
