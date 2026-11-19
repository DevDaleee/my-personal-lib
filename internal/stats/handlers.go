package stats

import (
	"errors"
	"log/slog"
	messages "my-personal-lib/internal/error"
	"my-personal-lib/internal/json"
	"my-personal-lib/internal/userctx"
	"net/http"
	"strconv"
	"strings"
)

type StatsHandler struct {
	service StatsServices
}

func NewHandler(service StatsServices) *StatsHandler {
	return &StatsHandler{
		service: service,
	}
}

func (h *StatsHandler) GetStatsDaily(w http.ResponseWriter, r *http.Request) {
	userID, ok := userctx.FromContext(r.Context())
	if !ok {
		slog.Error("Invalid User", "Error:", messages.ErrValidUser.Error(), "user_id", userID.String())
		json.WriteError(w, http.StatusInternalServerError, messages.ErrInternalServerError.Error())
		return
	}

	raw := strings.TrimSpace(r.URL.Query().Get("days"))
	days := 7
	if raw != "" {
		day, err := strconv.Atoi(raw)
		if err != nil {
			json.WriteError(w, http.StatusBadRequest, messages.ErrInvalidDays.Error())
			return
		}
		days = day
	}
	listBook, err := h.service.GetStatsDaily(r.Context(), userID, days)
	if err != nil {
		if errors.Is(err, messages.ErrInvalidDays) {
			json.WriteError(w, http.StatusBadRequest, messages.ErrInvalidDays.Error())
			return
		}
		slog.Error("Get status daily failed", "error", err.Error(), "days", days)
		json.WriteError(w, http.StatusInternalServerError, messages.ErrInternalServerError.Error())
		return
	}
	json.Write(w, http.StatusOK, listBook)
}
