-- +goose Up
CREATE TABLE progress_updates(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  book_id UUID NOT NULL,
  previous_page INTEGER NOT NULL CHECK (previous_page >= 0),
  current_page INTEGER NOT NULL CHECK (current_page >= 0),
  status TEXT NOT NULL CHECK (status IN('wanna_read', 'reading', 'read')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT fk_book_id FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS progress_updates;
