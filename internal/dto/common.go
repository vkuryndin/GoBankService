package dto

type ErrorResponse struct {
	Error   string `json:"error"`
	Details any    `json:"details,omitempty"`
}

type MessageResponse struct {
	Message string `json:"message"`
}
