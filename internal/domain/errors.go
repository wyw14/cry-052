package domain

type FaultCode string

const (
	FaultMissingGovernanceResource FaultCode = "GOVERNANCE_RESOURCE_NOT_FOUND"
	FaultGovernanceConflict        FaultCode = "GOVERNANCE_CONFLICT"
	FaultBatchTransition           FaultCode = "BATCH_TRANSITION_INVALID"
	FaultCatalogVersion            FaultCode = "CATALOG_VERSION_CONFLICT"
	FaultSourceOverwrite           FaultCode = "SOURCE_OVERWRITE_FORBIDDEN"
	FaultPreviewConfirmation       FaultCode = "PREVIEW_CONFIRMATION_REQUIRED"
	FaultRolePermission            FaultCode = "ROLE_PERMISSION_DENIED"
	FaultBatchCancellation         FaultCode = "BATCH_ALREADY_CANCELLED"
	FaultIdempotencyInput          FaultCode = "IDEMPOTENCY_INPUT_MISMATCH"
	FaultInputSnapshot             FaultCode = "INPUT_SNAPSHOT_CHANGED"
)

type Fault struct {
	Code    FaultCode
	Message string
}

func (f *Fault) Error() string { return f.Message }

var (
	ErrNotFound             = &Fault{Code: FaultMissingGovernanceResource, Message: "governance resource not found"}
	ErrConflict             = &Fault{Code: FaultGovernanceConflict, Message: "governance operation conflicts with current state"}
	ErrInvalidTransition    = &Fault{Code: FaultBatchTransition, Message: "batch state transition is not allowed"}
	ErrVersionConflict      = &Fault{Code: FaultCatalogVersion, Message: "governance record version has changed"}
	ErrSourceOverwrite      = &Fault{Code: FaultSourceOverwrite, Message: "masking output cannot overwrite its source table"}
	ErrPreviewRequired      = &Fault{Code: FaultPreviewConfirmation, Message: "a confirmed masking preview is required"}
	ErrPermissionDenied     = &Fault{Code: FaultRolePermission, Message: "governance role is not permitted"}
	ErrAlreadyCancelled     = &Fault{Code: FaultBatchCancellation, Message: "masking batch is already cancelled"}
	ErrIdempotencyReuse     = &Fault{Code: FaultIdempotencyInput, Message: "idempotency key belongs to different masking input"}
	ErrInputSnapshotChanged = &Fault{Code: FaultInputSnapshot, Message: "source snapshot changed after masking preview"}
)

type FieldViolation struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationError struct {
	Code       string           `json:"code"`
	Message    string           `json:"message"`
	Violations []FieldViolation `json:"violations,omitempty"`
}

func (e *ValidationError) Error() string { return e.Message }

func NewValidationError(message string, violations ...FieldViolation) error {
	return &ValidationError{Code: "VALIDATION_FAILED", Message: message, Violations: violations}
}
