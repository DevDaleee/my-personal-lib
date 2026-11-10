package userctx

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	messages "my-personal-lib/internal/error"
	"my-personal-lib/internal/json"
)

// HeaderName é o cabeçalho HTTP que traz o user_id.
const HeaderName = "X-User-ID"

// ctxKey é o tipo da chave usada para guardar o user_id no context.
// Por ser um tipo próprio e não exportado, nenhum outro pacote consegue
// criar uma chave igual e sobrescrever (ou ler por engano) o nosso valor.
type ctxKey struct{}

// Middleware lê o cabeçalho X-User-ID, converte para UUID e guarda no context.
// Se o cabeçalho faltar ou não for um UUID válido, responde 400 e
// não chama o próximo handler (a requisição para aqui).
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get(HeaderName)
		if raw == "" {
			json.WriteError(w, http.StatusBadRequest, messages.ErrHeaderField.Error())
			return
		}

		// Scan converte o texto em UUID e devolve erro se o formato for inválido.
		var userID pgtype.UUID
		if err := userID.Scan(raw); err != nil {
			json.WriteError(w, http.StatusBadRequest, messages.ErrValidHeaderField.Error())
			return
		}

		// Contextos são imutáveis: WithValue devolve um context novo,
		// e r.WithContext devolve uma cópia da requisição usando esse context.
		ctx := context.WithValue(r.Context(), ctxKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// FromContext devolve o user_id guardado pelo Middleware.
// O bool é false se a requisição não passou pelo Middleware.
func FromContext(ctx context.Context) (pgtype.UUID, bool) {
	userID, ok := ctx.Value(ctxKey{}).(pgtype.UUID)
	return userID, ok
}
