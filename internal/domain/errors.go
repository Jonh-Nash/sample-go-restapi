package domain

type ValidationReason string

const (
	ValidationReasonCredentialRequired ValidationReason = "credential_required"
	ValidationReasonInputLength        ValidationReason = "input_length"
	ValidationReasonInvalidPattern     ValidationReason = "invalid_pattern"
	ValidationReasonProfileRequired    ValidationReason = "profile_required"
	ValidationReasonProfileConstraint  ValidationReason = "profile_constraint"
)

type ErrValidation struct {
	Reason ValidationReason
}

func (e *ErrValidation) Error() string { return string(e.Reason) }
