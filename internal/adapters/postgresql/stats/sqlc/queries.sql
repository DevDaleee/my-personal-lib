-- ============ Consumer (as 3 rodam na mesma transação) ============

-- name: InsertReadingEvent :execrows
-- Grava o evento bruto. Se o event_id já existe, não faz nada (idempotência).
-- Linhas afetadas: 1 = evento novo (seguir para os resumos);
--                  0 = duplicado (pular o resto, commit e confirmar a mensagem).
-- pages_read = max(current_page - previous_page, 0) e event_date (dia em UTC)
-- são calculados no Go e passados como parâmetro.
INSERT INTO reading_events (
    event_id, book_id, user_id, previous_page, current_page,
    pages_read, status, event_date, occurred_at
) VALUES (
    @event_id, @book_id, @user_id, @previous_page, @current_page,
    @pages_read, @status, @event_date, @occurred_at
)
ON CONFLICT (event_id) DO NOTHING;

-- name: UpsertDailySummary :exec
-- Cria a linha do dia ou, se já existe, soma as páginas e mais 1 atualização.
-- EXCLUDED é a linha que tentamos inserir; daily_reading_summary.* é a que já está lá.
-- Só chamar se o InsertReadingEvent devolveu 1.
INSERT INTO daily_reading_summary (user_id, day, pages_read, updates)
VALUES (@user_id, @day, @pages_read, 1)
ON CONFLICT (user_id, day) DO UPDATE
SET pages_read = daily_reading_summary.pages_read + EXCLUDED.pages_read,
    updates    = daily_reading_summary.updates + 1;

-- name: InsertFinishedBook :exec
-- Registra a conclusão do livro. Chamar só quando status = 'read'.
-- Se o livro já foi concluído antes, mantém a primeira data.
INSERT INTO finished_books (user_id, book_id, finished_on)
VALUES (@user_id, @book_id, @finished_on)
ON CONFLICT (user_id, book_id) DO NOTHING;

-- ============ Endpoints GET /stats/* ============

-- name: ListDailyStats :many
-- GET /stats/daily. start_day é calculado no Go (ex.: hoje - 30 dias).
-- Dias sem leitura não aparecem (não existe linha para eles).
SELECT day, pages_read, updates
FROM daily_reading_summary
WHERE user_id = @user_id AND day >= @start_day
ORDER BY day;

-- name: ListMonthlyFinished :many
-- GET /stats/monthly. Livros concluídos por mês desde start_day (calculado no Go).
-- date_trunc devolve data com hora; o ::date deixa só a data.
-- Os apelidos (AS) viram os nomes dos campos na struct gerada pelo sqlc.
SELECT date_trunc('month', finished_on)::date AS month,
       count(*) AS finished_count
FROM finished_books
WHERE user_id = @user_id AND finished_on >= @start_day
GROUP BY month
ORDER BY month;

-- name: ListReadingDays :many
-- GET /stats/streak. Dias com leitura, do mais recente para o mais antigo.
-- A sequência de dias seguidos é calculada no Go.
SELECT day
FROM daily_reading_summary
WHERE user_id = @user_id AND pages_read > 0
ORDER BY day DESC;
