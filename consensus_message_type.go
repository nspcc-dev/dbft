package dbft

import "fmt"

// MessageType is a type for dBFT consensus messages.
type MessageType byte

// 7 following constants enumerate all possible type of consensus message.
const (
	ChangeViewType      MessageType = 0x00
	PrepareRequestType  MessageType = 0x20
	PrepareResponseType MessageType = 0x21
	PreCommitType       MessageType = 0x31
	CommitType          MessageType = 0x30
	RecoveryRequestType MessageType = 0x40
	RecoveryMessageType MessageType = 0x41
)

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
	case PreCommitType:
		return "PreCommit"
	case RecoveryRequestType:
		return "RecoveryRequest"
	case RecoveryMessageType:
		return "RecoveryMessage"
	default:
		return fmt.Sprintf("UNKNOWN(%02x)", byte(m))
	}
}
