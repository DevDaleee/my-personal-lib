package auth

import (
	"errors"
	"log/slog"
	messages "my-personal-lib/internal/error"
	"my-personal-lib/internal/json"
	"my-personal-lib/internal/userctx"
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

	user, err := h.service.Register(r.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		if errors.Is(err, messages.ErrInvalidEmail) || errors.Is(err, messages.ErrWeakPassword) || errors.Is(err, messages.ErrMissingName) {
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

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.Read(r, &req); err != nil {
		json.WriteError(w, http.StatusBadRequest, messages.ErrMissingFields.Error())
		return
	}

	token, userID, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, messages.ErrInvalidCredentials) {
			json.WriteError(w, http.StatusUnauthorized, err.Error())
			return
		}
		slog.Error("login failed", "error", err)
		json.WriteError(w, http.StatusInternalServerError, messages.ErrInternalServerError.Error())
		return
	}

	json.Write(w, http.StatusOK, loginResponse{Token: token, UserID: userID})
}

func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := userctx.FromContext(r.Context())
	if !ok {
		slog.Error("Invalid User", "error", messages.ErrValidUser.Error(), "user_id", userID.String())
		json.WriteError(w, http.StatusInternalServerError, messages.ErrInternalServerError.Error())
		return
	}

	user, err := h.service.GetMe(r.Context(), userID)
	if err != nil {
		if errors.Is(err, messages.ErrUserNotFound) {
			slog.Error("user not found", "error", messages.ErrUserNotFound.Error(), "user_id", userID.String())
			json.WriteError(w, http.StatusNotFound, messages.ErrUserNotFound.Error())
			return
		}
		slog.Error("failed getting user", "error", err)
		json.WriteError(w, http.StatusInternalServerError, messages.ErrInternalServerError.Error())
		return
	}
	json.Write(w, http.StatusOK, user)
}
