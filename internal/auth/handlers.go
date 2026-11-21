package auth

import (
	"errors"
	"log/slog"
	messages "my-personal-lib/internal/error"
	"my-personal-lib/internal/json"
	"net/http"
)

type AuthHandler struct {
	service AuthService
}

func NewHandler(service AuthService) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.Read(r, &req); err != nil {
		json.WriteError(w, http.StatusBadRequest, messages.ErrMissingFields.Error())
		return
	}

	user, err := h.service.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, messages.ErrInvalidEmail) || errors.Is(err, messages.ErrWeakPassword) {
			json.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, messages.ErrEmailTaken) {
			json.WriteError(w, http.StatusConflict, err.Error())
			return
		}
		slog.Error("fail register user", "error", err)
		json.WriteError(w, http.StatusInternalServerError, messages.ErrInternalServerError.Error())
		return
	}
	json.Write(w, http.StatusCreated, user)
}
