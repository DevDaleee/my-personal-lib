package auth

import (
	"context"
	"errors"
	"fmt"
	repo "my-personal-lib/internal/adapters/postgresql/books/sqlc"
	messages "my-personal-lib/internal/error"
	"net/mail"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(ctx context.Context, email string, password string) (repo.CreateUserRow, error)
	Login(ctx context.Context, email string, password string) (string, error)
}

type svc struct {
	repo      *repo.Queries
	jwtSecret []byte
}

func NewService(repo *repo.Queries, jwtSecret []byte) AuthService {
	return &svc{repo: repo, jwtSecret: jwtSecret}
}

func (s *svc) Register(ctx context.Context, email string, password string) (repo.CreateUserRow, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return repo.CreateUserRow{}, messages.ErrInvalidEmail
	}
	if len(password) < 8 || len(password) > 72 {
		return repo.CreateUserRow{}, messages.ErrWeakPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return repo.CreateUserRow{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, repo.CreateUserParams{
		Email:        email,
		PasswordHash: string(hash),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return repo.CreateUserRow{}, messages.ErrEmailTaken
		}
		return repo.CreateUserRow{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (s *svc) Login(ctx context.Context, email string, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", messages.ErrInvalidCredentials
		}
		return "", fmt.Errorf("get user: %w", err)
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", messages.ErrInvalidCredentials
	}
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sing token: %w", err)
	}

	return signed, nil
}
