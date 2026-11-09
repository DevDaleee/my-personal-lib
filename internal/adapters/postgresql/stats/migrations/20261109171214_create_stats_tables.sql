-- +goose Up
-- Um evento ReadingProgressUpdated recebido do NATS por linha.
-- event_id vem do evento (sem DEFAULT): a PK é o que garante a idempotência.
-- book_id e user_id são cópias lógicas do books_db, sem FK.
CREATE TABLE reading_events(
  event_id UUID PRIMARY KEY,
  book_id UUID NOT NULL,
  user_id UUID NOT NULL,
  previous_page INTEGER NOT NULL CHECK (previous_page >= 0),
  current_page INTEGER NOT NULL CHECK (current_page >= 0),
  pages_read INTEGER NOT NULL CHECK (pages_read >= 0),
  status TEXT NOT NULL CHECK (status IN('wanna_read', 'reading', 'read')),
  event_date DATE NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL,
  received_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Resumo por usuário por dia. A PK composta é o alvo do upsert
-- e também serve de índice para as consultas por usuário e intervalo de dias.
CREATE TABLE daily_reading_summary(
  user_id UUID NOT NULL,
  day DATE NOT NULL,
  pages_read INTEGER NOT NULL DEFAULT 0 CHECK (pages_read >= 0),
  updates INTEGER NOT NULL DEFAULT 0 CHECK (updates >= 0),
  PRIMARY KEY (user_id, day)
);

-- Livros concluídos. A PK impede contar o mesmo livro duas vezes
-- (fica registrada a primeira conclusão).
CREATE TABLE finished_books(
  user_id UUID NOT NULL,
  book_id UUID NOT NULL,
  finished_on DATE NOT NULL,
  PRIMARY KEY (user_id, book_id)
);

-- +goose Down
DROP TABLE IF EXISTS finished_books;
DROP TABLE IF EXISTS daily_reading_summary;
DROP TABLE IF EXISTS reading_events;
