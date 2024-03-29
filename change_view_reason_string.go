// Code generated by "stringer -type=ChangeViewReason -linecomment"; DO NOT EDIT.

package dbft

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[CVTimeout-0]
	_ = x[CVChangeAgreement-1]
	_ = x[CVTxNotFound-2]
	_ = x[CVTxRejectedByPolicy-3]
	_ = x[CVTxInvalid-4]
	_ = x[CVBlockRejectedByPolicy-5]
	_ = x[CVUnknown-255]
}

const (
	_ChangeViewReason_name_0 = "TimeoutChangeAgreementTxNotFoundTxRejectedByPolicyTxInvalidBlockRejectedByPolicy"
	_ChangeViewReason_name_1 = "Unknown"
)

var (
	_ChangeViewReason_index_0 = [...]uint8{0, 7, 22, 32, 50, 59, 80}
)

func (i ChangeViewReason) String() string {
	switch {
	case 0 <= i && i <= 5:
		return _ChangeViewReason_name_0[_ChangeViewReason_index_0[i]:_ChangeViewReason_index_0[i+1]]
	case i == 255:
		return _ChangeViewReason_name_1
	default:
		return "ChangeViewReason(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}
