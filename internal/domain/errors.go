package domain

// AppError is a domain error carrying a stable machine-readable code, used
// directly to build the {"error":{"code":...,"message":...}} API response
// without ever leaking internal details (stack traces, SQL, file paths).
type AppError struct {
	Code    string
	Message string
	Status  int // HTTP status to use; 0 means "let the caller decide" (defaults to 400)
}

func (e *AppError) Error() string { return e.Message }

func NewError(code, message string, status int) *AppError {
	return &AppError{Code: code, Message: message, Status: status}
}

var (
	ErrNotFound         = NewError("NOT_FOUND", "Resource not found", 404)
	ErrUnauthorized     = NewError("UNAUTHORIZED", "Authentication required", 401)
	ErrForbidden        = NewError("FORBIDDEN", "You do not have permission to perform this action", 403)
	ErrInvalidInput     = NewError("INVALID_INPUT", "Invalid input", 400)
	ErrConflict         = NewError("CONFLICT", "Conflict", 409)
	ErrInsufficientFund = NewError("INSUFFICIENT_PAYMENT", "Payment does not cover the total due", 400)
)
