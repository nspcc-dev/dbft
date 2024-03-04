package dbft

//go:generate stringer -type=ChangeViewReason -linecomment

// ChangeViewReason represents a view change reason code.
type ChangeViewReason byte

// These constants define various reasons for view changing. They're following
// Neo 3 except the Unknown value which is left for compatibility with Neo 2.
const (
	CVTimeout               ChangeViewReason = 0x0  // Timeout
	CVChangeAgreement       ChangeViewReason = 0x1  // ChangeAgreement
	CVTxNotFound            ChangeViewReason = 0x2  // TxNotFound
	CVTxRejectedByPolicy    ChangeViewReason = 0x3  // TxRejectedByPolicy
	CVTxInvalid             ChangeViewReason = 0x4  // TxInvalid
	CVBlockRejectedByPolicy ChangeViewReason = 0x5  // BlockRejectedByPolicy
	CVUnknown               ChangeViewReason = 0xff // Unknown
)
