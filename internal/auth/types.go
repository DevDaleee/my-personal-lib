package auth

import "github.com/jackc/pgx/v5/pgtype"

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token  string      `json:"token"`
	UserID pgtype.UUID `json:"user_id"`
}
