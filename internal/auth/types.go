package auth

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
