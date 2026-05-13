package dto

type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterResponse struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token  string `json:"token"`
	UserID int64  `json:"-"`
}

type AuthCheckResponse struct {
	Authenticated bool   `json:"authenticated"`
	UserID        int64  `json:"user_id"`
	Email         string `json:"email"`
	Username      string `json:"username"`
	IsAdmin       bool   `json:"is_admin"`
}
