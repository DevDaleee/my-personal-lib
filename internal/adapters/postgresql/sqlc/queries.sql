-- name: CreateBook :one
INSERT INTO books (
    user_id, title, author, isbn, total_pages, rating,
    notes, cover_url, owned, started_at, finished_at
) VALUES (
    @user_id, @title, @author, @isbn, @total_pages, @rating,
    @notes, @cover_url, @owned, @started_at, @finished_at
)
RETURNING *;

-- name: ListBooks :many
SELECT * FROM books
WHERE user_id = @user_id
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY updated_at DESC;

-- name: GetBook :one
SELECT * FROM books
WHERE id = @id AND user_id = @user_id;

-- name: UpdateBook :one
UPDATE books
SET title       = @title,
    author      = @author,
    isbn        = @isbn,
    total_pages = @total_pages,
    cover_url   = @cover_url,
    owned       = @owned,
    updated_at  = now()
WHERE id = @id AND user_id = @user_id
RETURNING *;

-- name: UpdateProgress :one
-- O status é decidido no Go e passado como parâmetro.
-- started_at: preenchido só na primeira vez que o livro entra em 'reading'.
-- finished_at: preenchido ao virar 'read'; limpo se o livro sair de 'read'.
UPDATE books
SET current_page = @current_page,
    status       = @status,
    started_at   = COALESCE(
                       started_at,
                       CASE WHEN @status::text = 'reading' THEN now() END
                   ),
    finished_at  = CASE
                       WHEN @status::text = 'read' THEN COALESCE(finished_at, now())
                       ELSE NULL
                   END,
    updated_at   = now()
WHERE id = @id AND user_id = @user_id
RETURNING *;

-- name: SetRating :one
UPDATE books
SET rating     = @rating,
    updated_at = now()
WHERE id = @id AND user_id = @user_id
RETURNING *;

-- name: DeleteBook :execrows
DELETE FROM books
WHERE id = @id AND user_id = @user_id;

-- name: ListProgressUpdates :many
SELECT pu.* FROM progress_updates pu
JOIN books b ON b.id = pu.book_id
WHERE pu.book_id = @book_id AND b.user_id = @user_id
ORDER BY pu.created_at;

-- name: GetBookForUpdate :one
-- Igual ao GetBook, mas trava a linha até o fim da transação (FOR UPDATE).
-- Assim dois PATCH simultâneos não leem o mesmo current_page e o stats
-- não conta as mesmas páginas duas vezes. Só funciona dentro de uma transação.
SELECT * FROM books
WHERE id = @id AND user_id = @user_id
FOR UPDATE;

-- name: CreateProgressUpdate :one
-- Registra o histórico do progresso. id e created_at vêm do DEFAULT do banco:
-- o id vira o event_id e o created_at vira o updated_at do evento no NATS.
-- Roda na mesma transação do GetBookForUpdate e do UpdateProgress.
INSERT INTO progress_updates (
    book_id, previous_page, current_page, status
) VALUES (
    @book_id, @previous_page, @current_page, @status
)
RETURNING id, created_at;
