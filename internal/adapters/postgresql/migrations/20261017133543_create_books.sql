-- +goose Up
CREATE TABLE books(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  title TEXT NOT NULL,
  author TEXT NOT NULL,
  isbn TEXT,
  total_pages INTEGER NOT NULL CHECK (total_pages > 0),
  current_page INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'wanna_read' CHECK (status IN('wanna_read', 'reading', 'read')),
  rating INTEGER CHECK(rating > 0 AND rating <= 5),
  notes TEXT,
  cover_url TEXT,
  owned BOOLEAN NOT NULL DEFAULT false,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (current_page >= 0 AND current_page <= total_pages)
);
-- isbn unique for user (just when its filled up)
CREATE UNIQUE INDEX books_user_isbn_key ON books (user_id, isbn) WHERE isbn IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS books;
