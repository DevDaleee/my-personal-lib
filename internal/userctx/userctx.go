package userctx

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"

	messages "my-personal-lib/internal/error"
	"my-personal-lib/internal/json"
)

// ctxKey é o tipo da chave usada para guardar o user_id no context.
// Por ser um tipo próprio e não exportado, nenhum outro pacote consegue
// criar uma chave igual e sobrescrever (ou ler por engano) o nosso valor.
type ctxKey struct{}

// Middleware fabrica o middleware de autenticação usando a chave do JWT.
// Ele é uma função que devolve outra função porque o chi exige o formato
// func(http.Handler) http.Handler no r.Use, e o middleware precisa "lembrar"
// da chave (closure). Uso: r.Use(userctx.Middleware(secret)).
//
// O middleware lê "Authorization: Bearer <token>", confere a assinatura e a
// validade do token e guarda no context o user_id que está no campo "sub".
// Sem token, ou com token inválido/vencido, responde 401 e a requisição para aqui.
func Middleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// CutPrefix tira o "Bearer " do começo e diz se ele estava lá.
			tokenStr, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || tokenStr == "" {
				json.WriteError(w, http.StatusUnauthorized, messages.ErrMissingToken.Error())
				return
			}

			// ParseWithClaims confere a assinatura com a chave e o "exp" (validade).
			// WithValidMethods aceita só HS256: sem isso, um token com "alg": "none"
			// (sem assinatura) poderia ser aceito.
			claims := &jwt.RegisteredClaims{}
			_, err := jwt.ParseWithClaims(tokenStr, claims,
				func(t *jwt.Token) (any, error) { return secret, nil },
				jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			)
			if err != nil {
				json.WriteError(w, http.StatusUnauthorized, messages.ErrInvalidToken.Error())
				return
			}

			// O "sub" carrega o id do usuário (ver auth.Login).
			var userID pgtype.UUID
			if err := userID.Scan(claims.Subject); err != nil {
				json.WriteError(w, http.StatusUnauthorized, messages.ErrInvalidToken.Error())
				return
			}

			// Contextos são imutáveis: WithValue devolve um context novo,
			// e r.WithContext devolve uma cópia da requisição usando esse context.
			ctx := context.WithValue(r.Context(), ctxKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// FromContext devolve o user_id guardado pelo Middleware.
// O bool é false se a requisição não passou pelo Middleware.
func FromContext(ctx context.Context) (pgtype.UUID, bool) {
	userID, ok := ctx.Value(ctxKey{}).(pgtype.UUID)
	return userID, ok
}
