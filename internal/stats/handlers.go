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

// GetStatsStreak devolve a sequência atual de dias lendo.
//
// @Summary Sequência de dias lendo
// @Description Quantos dias seguidos, até hoje (America/Sao_Paulo), o usuário registrou leitura.
// @Tags stats
// @Produce json
// @Security BearerAuth
// @Success 200 {object} StreakResponse
// @Failure 401 {object} map[string]string "Token ausente, inválido ou vencido"
// @Failure 500 {object} map[string]string
// @Router /stats/streak [get]
func (h *StatsHandler) GetStatsStreak(w http.ResponseWriter, r *http.Request) {
	userID, ok := userctx.FromContext(r.Context())
	if !ok {
		slog.Error("Invalid User", "Error:", messages.ErrValidUser.Error(), "user_id", userID.String())
		json.WriteError(w, http.StatusInternalServerError, messages.ErrInternalServerError.Error())
		return
	}

	streak, err := h.service.GetStatsStreak(r.Context(), userID)
	if err != nil {
		slog.Error("Get streak failed", "error", err)
		json.WriteError(w, http.StatusInternalServerError, messages.ErrInternalServerError.Error())
		return
	}

	json.Write(w, http.StatusOK, StreakResponse{CurrentStreak: streak})

}

// GetStatsMonthly devolve os livros concluídos por mês.
//
// @Summary Livros concluídos por mês
// @Description Livros concluídos por mês nos últimos N meses: 2 (bimestral), 3 (trimestral), 6 (semestral) ou 12 (anual). Meses sem livro não aparecem.
// @Tags stats
// @Produce json
// @Security BearerAuth
// @Param months query int false "Período em meses" Enums(2, 3, 6, 12) default(12)
// @Success 200 {array} statsrepo.ListMonthlyFinishedRow
// @Failure 400 {object} map[string]string "Período inválido"
// @Failure 401 {object} map[string]string "Token ausente, inválido ou vencido"
// @Failure 500 {object} map[string]string
// @Router /stats/monthly [get]
func (h *StatsHandler) GetStatsMonthly(w http.ResponseWriter, r *http.Request) {
	userID, ok := userctx.FromContext(r.Context())
	if !ok {
		slog.Error("Invalid User", "Error:", messages.ErrValidUser.Error(), "user_id", userID.String())
		json.WriteError(w, http.StatusInternalServerError, messages.ErrInternalServerError.Error())
		return
	}

	raw := strings.TrimSpace(r.URL.Query().Get("months"))
	months := 12
	if raw != "" {
		month, err := strconv.Atoi(raw)
		if err != nil {
			json.WriteError(w, http.StatusBadRequest, messages.ErrInvalidMonthly.Error())
			return
		}
		months = month
	}
	listBook, err := h.service.GetStatsMonthly(r.Context(), userID, months)
	if err != nil {
		if errors.Is(err, messages.ErrInvalidMonthly) {
			json.WriteError(w, http.StatusBadRequest, messages.ErrInvalidMonthly.Error())
			return
		}
		slog.Error("Get status monthly failed", "error", err.Error(), "months", months)
		json.WriteError(w, http.StatusInternalServerError, messages.ErrInternalServerError.Error())
		return
	}
	json.Write(w, http.StatusOK, listBook)
}

// GetStatsDaily devolve as páginas lidas por dia.
//
// @Summary Páginas por dia
// @Description Páginas lidas por dia nos últimos N dias (America/Sao_Paulo). Dias sem leitura não aparecem.
// @Tags stats
// @Produce json
// @Security BearerAuth
// @Param days query int false "Período em dias" Enums(7, 15, 30) default(7)
// @Success 200 {array} statsrepo.ListDailyStatsRow
// @Failure 400 {object} map[string]string "Período inválido"
// @Failure 401 {object} map[string]string "Token ausente, inválido ou vencido"
// @Failure 500 {object} map[string]string
// @Router /stats/daily [get]
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
