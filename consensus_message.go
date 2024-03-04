package dbft

// ConsensusMessage is an interface for generic dBFT message.
type ConsensusMessage[H Hash, A Address] interface {
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
