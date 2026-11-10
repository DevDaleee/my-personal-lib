# Biblioteca Pessoal

App para acompanhar o progresso de leitura: cadastrar livros, registrar a página atual e ver estatísticas como páginas por dia, livros concluídos por mês e sequência de dias lendo.

É um **projeto acadêmico** para praticar **arquitetura de microserviços em Go**, com uma interface em **Flutter** rodando na web. O objetivo é aprender, então a prioridade é um projeto simples, bem feito e fácil de explicar.

> **Status:** em desenvolvimento (outubro/2026 a janeiro/2027). Veja o [roadmap](#roadmap) para saber o que já está pronto.

## Arquitetura

```
App Flutter (web; mobile depois)
        │  HTTP/JSON
        ▼
API Gateway (Nginx, porta 8080) ──/books/*──▶ books-service ──▶ books_db (Postgres)
        │                       ──/stats/*──▶ stats-service ──▶ stats_db (Postgres)
        └── serve o build web do Flutter em /

books-service ──publica ReadingProgressUpdated──▶ NATS ──▶ stats-service
```

- **Dois serviços, cada um com o seu banco.** Não há FK nem consulta entre os bancos.
- **Comunicação síncrona** por REST/JSON e **assíncrona** por eventos no NATS.
- **O app fala só com o Gateway**, nunca direto com os serviços, os bancos ou o NATS.
- **CORS fica no Nginx**, num lugar só.

| Componente | Responsabilidade | Banco |
|---|---|---|
| `books-service` | Livros do usuário e atualização do progresso de leitura. Publica o evento. | `books_db` |
| `stats-service` | Consome o evento e calcula as estatísticas. | `stats_db` |
| `gateway` (Nginx) | Roteia a API, aplica CORS e serve o app web. | — |
| `app` (Flutter) | Interface do usuário. | — |

## Stack

- **Backend:** Go 1.27, [chi v5](https://go-chi.io), [pgx v5](https://github.com/jackc/pgx), [sqlc](https://docs.sqlc.dev), [goose](https://pressly.github.io/goose/), `log/slog`, [nats.go](https://github.com/nats-io/nats.go)
- **Banco:** PostgreSQL 16
- **Frontend:** Flutter (web, preparado para virar mobile), Material 3, pacote `http`
- **Infra:** Docker Compose e Nginx

## Estrutura de pastas

```
my-personal-lib/
  go.mod                      # módulo Go único: my-personal-lib
  sqlc.yaml                   # configuração do sqlc
  docker-compose.yaml
  cmd/                        # books-service (main.go e rotas em api.go)
  internal/
    env/                      # leitura de variáveis de ambiente
    json/                     # helpers de JSON para as respostas
    adapters/postgresql/
      migrations/             # migrations do books_db (goose)
      sqlc/queries.sql        # queries do books (o código Go é gerado aqui)
      stats/
        migrations/           # migrations do stats_db (goose)
        sqlc/queries.sql      # queries do stats
  gateway/                    # (planejado) nginx.conf
  app/                        # (planejado) projeto Flutter
```

Os arquivos `.go` gerados pelo sqlc **não são editados à mão**. Para mudar o acesso a dados, altere o `queries.sql` ou as migrations e rode `sqlc generate`.

## Como rodar

### Pré-requisitos
- [Go 1.27+](https://go.dev/dl/)
- [Docker](https://docs.docker.com/get-docker/) com Docker Compose
- [goose](https://pressly.github.io/goose/installation/) (migrations)
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html) (geração do código de acesso ao banco)
- [Flutter](https://docs.flutter.dev/get-started) (só para o app, a partir da semana 8)

### Variáveis de ambiente
Crie um arquivo `.env` na raiz. Ele está no `.gitignore` e **nunca deve ser commitado**.

| Variável | Para que serve |
|---|---|
| `DB_USER`, `DB_PASS`, `DB_NAME` | Usuário, senha e banco do Postgres no Compose |
| `GOOSE_DBSTRING` | DSN do Postgres, usado pelo goose **e** pela API |
| `GOOSE_DRIVER` | Driver do goose (`postgres`) |
| `GOOSE_MIGRATION_DIR` | Pasta das migrations do books (`internal/adapters/postgresql/migrations`) |
| `PORT` | Endereço da API, no formato `:porta` |
| `NATS_URL` | (planejado) Endereço do NATS |

### Passo a passo (books-service)
```bash
docker compose up -d     # sobe o Postgres
goose up                 # aplica as migrations (lê GOOSE_* do .env/ambiente)
sqlc generate            # gera o código Go a partir das queries
go run ./cmd             # sobe a API
```

Para conferir se a API subiu, chame a rota de saúde no endereço definido em `PORT`. Hoje ela se chama `/health`; o nome final será `/healthz`.

### Comandos úteis
```bash
# Go
go test ./...
go vet ./...
gofmt -l .

# Banco
goose status
goose create nome_da_migration sql
sqlc compile             # valida as queries sem gerar arquivos

# Infra
docker compose down -v   # derruba tudo e apaga os volumes
```

## API

Respostas em JSON com campos em `snake_case`. Erros vêm com um corpo `{"error": "mensagem"}`.

### Identificação do usuário
Não há login. Toda requisição (exceto `/healthz`) precisa do cabeçalho **`X-User-ID`** com um UUID, e os serviços só devolvem e alteram os dados desse usuário. Se o cabeçalho faltar ou não for um UUID válido, a resposta é **400**. O `user_id` nunca vai no corpo.

```bash
curl -H "X-User-ID: 11111111-1111-1111-1111-111111111111" http://localhost:8080/books
```

> ⚠️ **Isto não é autenticação.** A identidade é simulada: qualquer cliente pode enviar qualquer UUID. Em produção, ela viria de um token validado no gateway; como a leitura fica isolada num middleware, só ele mudaria.

**books-service**
| Método e rota | O que faz |
|---|---|
| `POST /books` | Cadastra um livro |
| `GET /books?status=` | Lista os livros, com filtro opcional por status |
| `GET /books/{id}` | Detalhe de um livro |
| `PATCH /books/{id}/progress` | Atualiza a página atual e o status; publica o evento |
| `GET /healthz` | Saúde do serviço |

**stats-service**
| Método e rota | O que faz |
|---|---|
| `GET /stats/daily` | Páginas lidas por dia |
| `GET /stats/monthly` | Livros concluídos por mês |
| `GET /stats/streak` | Sequência atual de dias lendo |
| `GET /healthz` | Saúde do serviço |

## Evento `ReadingProgressUpdated`

Publicado pelo `books-service` no subject `books.progress.updated` do NATS, **depois** que o progresso é gravado (após o commit). É consumido pelo `stats-service`.

```json
{
  "event_id": "9c1e0a52-...",
  "event": "ReadingProgressUpdated",
  "book_id": "b3f1...",
  "user_id": "u1...",
  "previous_page": 100,
  "current_page": 120,
  "total_pages": 320,
  "status": "reading",
  "updated_at": "2026-10-20T14:30:00Z"
}
```

- **Consumer idempotente:** um evento repetido (mesmo `event_id`) é ignorado e não altera as estatísticas.
- **Uma transação por evento:** o registro do evento e a atualização dos resumos são gravados juntos, ou nada é gravado.
- **Consistência eventual:** logo depois de um `PATCH`, as estatísticas podem mostrar o valor antigo por um instante. Isso é esperado.

## Modelo de dados

**books_db**
- `books`: cópia pessoal de cada livro do usuário (título, autor, páginas, página atual, status `wanna_read` / `reading` / `read`, nota, datas de início e fim).
- `progress_updates`: histórico de cada atualização de progresso. O `id` é usado como `event_id` do evento.

**stats_db**
- `reading_events`: cada evento recebido. A chave primária `event_id` garante a idempotência.
- `daily_reading_summary`: páginas lidas e número de atualizações por usuário por dia.
- `finished_books`: livros concluídos, com a data da primeira conclusão.

As regras de status e de intervalo de página ficam também no banco (`CHECK`), além da validação no Go.

## Roadmap

| Semana | Foco | |
|---|---|---|
| 1 | Go básico + servidor HTTP com chi | 🚧 servidor com chi pronto; faltam `POST /books` e `GET /books` |
| 2 | Postgres, pgx e primeiras migrations | 🚧 migrations e queries prontas; falta o CRUD usar o banco |
| 3 | `PATCH /books/{id}/progress`, validações e status | ⏳ |
| 4 | Testes, organização dos handlers, config por env | ⏳ |
| 5 | Dockerfile + Docker Compose (books + Postgres) | ⏳ |
| 6 | `stats-service` e `stats_db` + endpoints `/stats/*` | 🚧 migrations e queries prontas |
| 7 | NATS: publisher, consumer e idempotência | ⏳ |
| 8 | Gateway Nginx + `flutter create` | ⏳ |
| 9 | Flutter: cliente HTTP, modelos e tela "Minha biblioteca" | ⏳ |
| 10 | Flutter: telas "Novo livro" e "Detalhe" | ⏳ |
| 11 | Flutter: tela de estatísticas + slog, `/healthz` e graceful shutdown | ⏳ |
| 12 | Build web no Nginx, README final e apresentação | ⏳ |

### Fora do escopo
Autenticação, Kubernetes, Kafka, gRPC, publicação nas lojas de apps, modo offline e recursos sociais.
