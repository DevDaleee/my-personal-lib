package auth

import (
	"context"
	"errors"
	"fmt"
	repo "my-personal-lib/internal/adapters/postgresql/books/sqlc"
	messages "my-personal-lib/internal/error"
	"net/mail"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(ctx context.Context, email string, password string) (repo.CreateUserRow, error)
}

type svc struct {
	repo *repo.Queries
}

func NewService(repo *repo.Queries) AuthService {
	return &svc{repo: repo}
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
