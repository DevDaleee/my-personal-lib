-- +goose Up
-- Contas de usuário (login com e-mail e senha, decisão de 20/11/2026).
-- O e-mail é guardado sempre em minúsculas (o Go normaliza antes de gravar e de buscar),
-- por isso um UNIQUE simples basta para impedir duas contas com o mesmo e-mail.
-- A senha nunca é guardada: só o hash bcrypt.
CREATE TABLE users(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE CHECK (email = lower(email) AND email <> ''),
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- books.user_id aponta logicamente para users.id, sem FK: os livros criados antes
-- do login (com UUIDs de teste) continuam válidos.

-- +goose Down
DROP TABLE IF EXISTS users;
