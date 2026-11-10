package messages

import "errors"

var (
	ErrValidUser           = errors.New("Usuário inválido")
	ErrEmptyTitle          = errors.New("Título não pode ser vazio")
	ErrEmptyAuthor         = errors.New("Autor não pode ser vazio")
	ErrPageTotal           = errors.New("O total de páginas deve ser maior que 0")
	ErrRating              = errors.New("A pontuação de um livro tem que estar entre 1 e 5")
	ErrISBN                = errors.New("Você já tem um livro com esse ISBN")
	ErrInternalServerError = errors.New("Erro interno do servidor")
	ErrUserIDMissed        = errors.New("O campo user_id está vazio")
	ErrMissingFields       = errors.New("Campos inválidos")
	ErrHeaderField         = errors.New("O cabeçalho X-User-ID é obrigatório")
	ErrValidHeaderField    = errors.New("O cabeçalho X-User-ID deve ser um UUID válido")
)
