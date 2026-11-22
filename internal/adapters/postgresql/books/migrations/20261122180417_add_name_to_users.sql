-- +goose Up
-- Nome do usuário, obrigatório no cadastro.
-- O CHECK impede nome vazio ou só com espaços (btrim remove os espaços das pontas).
-- Sem DEFAULT: a tabela users estava vazia quando esta migration foi criada,
-- então não há linhas antigas para preencher.
ALTER TABLE users
  ADD COLUMN name TEXT NOT NULL CHECK (btrim(name) <> '');

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS name;
