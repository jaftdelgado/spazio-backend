package shared

// ErrorResponse is the standard error structure for API responses used by Swagger.
type ErrorResponse struct {
	Code    int    `json:"code" example:"404"`
	Message string `json:"message" example:"Resource not found"`
}
