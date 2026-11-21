package stats

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// testSecret é a chave do JWT usada só nos testes deste pacote.
var testSecret = []byte("chave-de-teste-com-mais-de-32-caracteres")

// bearer monta o valor do cabeçalho Authorization com um token válido para userID,
// do mesmo jeito que o auth.Login gera (HS256, user_id no "sub").
func bearer(t *testing.T, userID string) string {
	t.Helper()
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(testSecret)
	if err != nil {
		t.Fatalf("não consegui assinar o token de teste: %v", err)
	}
	return "Bearer " + token
}
