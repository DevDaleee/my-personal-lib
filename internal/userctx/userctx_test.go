package userctx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	messages "my-personal-lib/internal/error"
)

var testSecret = []byte("chave-de-teste-com-mais-de-32-caracteres")

const testUserID = "3d8be433-19f8-4963-b427-61e203353630"

// signToken monta um token como o auth.Login faz, com o método, a chave e as claims que o teste quiser.
func signToken(t *testing.T, method jwt.SigningMethod, key any, claims jwt.RegisteredClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("não consegui assinar o token de teste: %v", err)
	}
	return s
}

func validClaims() jwt.RegisteredClaims {
	now := time.Now()
	return jwt.RegisteredClaims{
		Subject:   testUserID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}
}

func TestMiddleware(t *testing.T) {
	expired := validClaims()
	expired.IssuedAt = jwt.NewNumericDate(time.Now().Add(-48 * time.Hour))
	expired.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-24 * time.Hour))

	notUUID := validClaims()
	notUUID.Subject = "ana@example.com"

	noSubject := validClaims()
	noSubject.Subject = ""

	tests := []struct {
		name          string
		authorization string // valor do cabeçalho; vazio = sem cabeçalho

		wantStatus     int
		wantError      string
		wantNextCalled bool
	}{
		{
			name:           "token válido passa e o user_id chega ao handler",
			authorization:  "Bearer " + signToken(t, jwt.SigningMethodHS256, testSecret, validClaims()),
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
		},
		{
			name:       "sem o cabeçalho Authorization devolve 401",
			wantStatus: http.StatusUnauthorized,
			wantError:  messages.ErrMissingToken.Error(),
		},
		{
			name:          "Authorization sem o prefixo Bearer devolve 401",
			authorization: signToken(t, jwt.SigningMethodHS256, testSecret, validClaims()),
			wantStatus:    http.StatusUnauthorized,
			wantError:     messages.ErrMissingToken.Error(),
		},
		{
			name:          "Bearer vazio devolve 401",
			authorization: "Bearer ",
			wantStatus:    http.StatusUnauthorized,
			wantError:     messages.ErrMissingToken.Error(),
		},
		{
			name:          "texto que não é JWT devolve 401",
			authorization: "Bearer isto-nao-e-um-token",
			wantStatus:    http.StatusUnauthorized,
			wantError:     messages.ErrInvalidToken.Error(),
		},
		{
			name:          "token vencido devolve 401",
			authorization: "Bearer " + signToken(t, jwt.SigningMethodHS256, testSecret, expired),
			wantStatus:    http.StatusUnauthorized,
			wantError:     messages.ErrInvalidToken.Error(),
		},
		{
			name:          "token assinado com outra chave devolve 401",
			authorization: "Bearer " + signToken(t, jwt.SigningMethodHS256, []byte("outra-chave-com-mais-de-32-caracteres!!"), validClaims()),
			wantStatus:    http.StatusUnauthorized,
			wantError:     messages.ErrInvalidToken.Error(),
		},
		{
			name:          `token com "alg": "none" (sem assinatura) devolve 401`,
			authorization: "Bearer " + signToken(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims()),
			wantStatus:    http.StatusUnauthorized,
			wantError:     messages.ErrInvalidToken.Error(),
		},
		{
			name:          "token com outro algoritmo HMAC (HS512) devolve 401",
			authorization: "Bearer " + signToken(t, jwt.SigningMethodHS512, testSecret, validClaims()),
			wantStatus:    http.StatusUnauthorized,
			wantError:     messages.ErrInvalidToken.Error(),
		},
		{
			name:          "sub que não é UUID devolve 401",
			authorization: "Bearer " + signToken(t, jwt.SigningMethodHS256, testSecret, notUUID),
			wantStatus:    http.StatusUnauthorized,
			wantError:     messages.ErrInvalidToken.Error(),
		},
		{
			name:          "token sem sub devolve 401",
			authorization: "Bearer " + signToken(t, jwt.SigningMethodHS256, testSecret, noSubject),
			wantStatus:    http.StatusUnauthorized,
			wantError:     messages.ErrInvalidToken.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// O "próximo handler" só anota se foi chamado e qual user_id recebeu.
			nextCalled := false
			var gotUserID string
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				id, ok := FromContext(r.Context())
				if !ok {
					t.Error("FromContext não achou o user_id")
				}
				gotUserID = id.String()
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/books", nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			rec := httptest.NewRecorder()

			Middleware(testSecret)(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, quero %d (corpo: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if nextCalled != tt.wantNextCalled {
				t.Fatalf("próximo handler chamado = %v, quero %v", nextCalled, tt.wantNextCalled)
			}

			if tt.wantNextCalled {
				if gotUserID != testUserID {
					t.Errorf("user_id no context = %q, quero %q", gotUserID, testUserID)
				}
				return
			}

			var body struct {
				Error string `json:"error"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("corpo de erro não é JSON: %v", err)
			}
			if body.Error != tt.wantError {
				t.Errorf("error = %q, quero %q", body.Error, tt.wantError)
			}
		})
	}
}

// Uma requisição que não passou pelo middleware não tem user_id.
func TestFromContextWithoutMiddleware(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, ok := FromContext(req.Context()); ok {
		t.Error("FromContext devolveu ok=true sem o middleware")
	}
}
