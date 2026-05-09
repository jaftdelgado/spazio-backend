package shared

// ErrorResponse represents a standard error payload for API responses.
type ErrorResponse struct {
	Error string `json:"error" example:"error message"`
}
