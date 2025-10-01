package domain

// ValidationReason はドメインルール違反を識別するコード。
// プレゼンテーション層のメッセージとは分離して扱う。
type ValidationReason string

const (
	ValidationReasonCredentialRequired ValidationReason = "credential_required"
	ValidationReasonInputLength        ValidationReason = "input_length"
	ValidationReasonInvalidPattern     ValidationReason = "invalid_pattern"
	ValidationReasonProfileRequired    ValidationReason = "profile_required"
	ValidationReasonProfileConstraint  ValidationReason = "profile_constraint"
)

// ErrValidation はドメインのバリデーションルール違反を表す。
type ErrValidation struct {
	Reason ValidationReason
}

func (e *ErrValidation) Error() string { return string(e.Reason) }
