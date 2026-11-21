[GO_BADGE]: https://img.shields.io/badge/Go-1.27-00ADD8?style=for-the-badge&logo=go&logoColor=white
[POSTGRES_BADGE]: https://img.shields.io/badge/PostgreSQL-16-4169E1?style=for-the-badge&logo=postgresql&logoColor=white
[NATS_BADGE]: https://img.shields.io/badge/NATS-JetStream-27AAE1?style=for-the-badge&logo=natsdotio&logoColor=white
[DOCKER_BADGE]: https://img.shields.io/badge/Docker-Compose-2496ED?style=for-the-badge&logo=docker&logoColor=white
[NGINX_BADGE]: https://img.shields.io/badge/Nginx-Gateway-009639?style=for-the-badge&logo=nginx&logoColor=white
[JWT_BADGE]: https://img.shields.io/badge/JWT-Auth-000?style=for-the-badge&logo=jsonwebtokens&logoColor=white
[FLUTTER_BADGE]: https://img.shields.io/badge/Flutter-Web-02569B?style=for-the-badge&logo=flutter&logoColor=white
[AWS_BADGE]: https://img.shields.io/badge/AWS-EC2-%23FF9900.svg?style=for-the-badge&logo=amazon-aws&logoColor=white
[AZURE_BADGE]: https://img.shields.io/badge/Azure-VM-0078D4?style=for-the-badge&logo=microsoftazure&logoColor=white

<h1 align="center" style="font-weight: bold;">Biblioteca Pessoal 📚</h1>

<p align="center">

![go][GO_BADGE]
![postgres][POSTGRES_BADGE]
![nats][NATS_BADGE]
![docker][DOCKER_BADGE]
![nginx][NGINX_BADGE]
![jwt][JWT_BADGE]
![flutter][FLUTTER_BADGE]
![aws][AWS_BADGE]
![azure][AZURE_BADGE]

</p>

<details open="open">
<summary>Sumário</summary>

- [🏛️ Arquitetura](#architecture)
- [💻 Tecnologias](#technologies)
- [🚀 Primeiros passos](#started)
  - [Pré-requisitos](#prerequisites)
  - [Clonando](#cloning)
  - [Variáveis de ambiente](#environment-variables)
  - [Iniciando](#starting)
- [📍 Endpoints da API](#routes)
  - [POST /auth/register](#post-register-detail)
  - [POST /auth/login](#post-login-detail)
  - [POST /books](#post-books-detail)
  - [GET /books](#get-books-detail)
  - [GET /books/{id}](#get-book-detail)
  - [PATCH /books/{id}/progress](#patch-progress-detail)
  - [GET /stats/daily](#get-daily-detail)
  - [GET /stats/monthly](#get-monthly-detail)
  - [GET /stats/streak](#get-streak-detail)
- [📨 Evento ReadingProgressUpdated](#event)
- [🗺️ Roadmap](#roadmap)
- [🤝 Colaboradores](#colab)
- [📫 Contribuindo](#contribute)

</details>

<p align="center">
  <b>App para acompanhar o progresso de leitura: cadastre seus livros, registre a página atual e veja estatísticas como páginas lidas por dia, livros concluídos por mês e sequência de dias lendo.</b>
</p>

<p align="center">
  Projeto acadêmico para praticar <b>arquitetura de microsserviços em Go</b>, com interface em <b>Flutter</b> na web.
</p>

> **Status:** back-end completo, com login (novembro/2026). Próximos passos: app Flutter mínimo e deploy na nuvem.

<h2 id="architecture">🏛️ Arquitetura</h2>

```
App Flutter (web; mobile depois)
        │  HTTP/JSON + Authorization: Bearer <token>
        ▼
API Gateway (Nginx, porta 8080) ──/auth, /books──▶ books-service ──▶ books_db (Postgres)
        │                       ──/stats───────▶ stats-service ──▶ stats_db (Postgres)
        └── serve o build web do Flutter em / (planejado)

books-service ──publica ReadingProgressUpdated──▶ NATS (JetStream) ──▶ stats-service
```

| Componente | Responsabilidade | Banco |
|---|---|---|
| `books-service` | Contas de usuário (cadastro e login), livros e progresso de leitura. Publica o evento. | `books_db` |
| `stats-service` | Consome o evento e calcula as estatísticas. | `stats_db` |
| `gateway` (Nginx) | Roteia a API, aplica CORS e serve o app web. | — |
| `app` (Flutter) | Interface do usuário (repositório separado: `my-personal-lib-app`). | — |

- **Cada serviço tem seu próprio banco**: não há FK nem consulta entre os bancos.
- **Comunicação síncrona** por REST/JSON e **assíncrona** por eventos no NATS JetStream.
- **O app fala só com o gateway**, nunca direto com os serviços, os bancos ou o NATS. O CORS fica no Nginx.
- **Login com JWT**: o books-service emite o token, e os dois serviços o validam com a mesma chave (`JWT_SECRET`), sem conversar entre si.

<h2 id="technologies">💻 Tecnologias</h2>

- **Go 1.27**, com [chi v5](https://go-chi.io) (rotas), [pgx v5](https://github.com/jackc/pgx) (driver do Postgres), [sqlc](https://docs.sqlc.dev) (código tipado a partir do SQL), [goose](https://pressly.github.io/goose/) (migrations) e `log/slog` (logs)
- **PostgreSQL 16**, um banco por serviço
- **NATS 2.10 com JetStream** ([nats.go](https://github.com/nats-io/nats.go))
- **JWT** ([golang-jwt v5](https://github.com/golang-jwt/jwt)) + **bcrypt** (`golang.org/x/crypto`)
- **Nginx** como API gateway
- **Docker** e **Docker Compose**
- **Flutter** (web, preparado para virar mobile)
- **AWS EC2** e **Azure VM** para o deploy

<h2 id="started">🚀 Primeiros passos</h2>

Com Docker Compose, um único comando sobe tudo: os dois Postgres, as migrations, o NATS, os dois serviços e o gateway.

<h3 id="prerequisites">Pré-requisitos</h3>

- [Git](https://git-scm.com/)
- [Docker](https://docs.docker.com/get-docker/) com Docker Compose (o suficiente para rodar o projeto)
- Para desenvolver fora do Docker: [Go 1.27+](https://go.dev/dl/), [goose](https://pressly.github.io/goose/installation/) e [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html)

<h3 id="cloning">Clonando</h3>

```bash
git clone https://github.com/DevDaleee/my-personal-lib.git
cd my-personal-lib
```

<h3 id="environment-variables">Variáveis de ambiente</h3>

Use o `.env.example` como base para criar o seu `.env` e **troque as senhas e o `JWT_SECRET`**. O `.env` está no `.gitignore` e nunca deve ser commitado.

```bash
cp .env.example .env
```

```yaml
# DSNs dos bancos (usados fora do Compose; no Compose o host é o nome do serviço)
GOOSE_DBSTRING=host=localhost user=db_user password=change-me dbname=db_name sslmode=disable port=5432
STATS_DBSTRING=host=localhost user=db_user password=change-me dbname=db_name sslmode=disable port=5433
GOOSE_DRIVER=postgres
GOOSE_MIGRATION_DIR=./internal/adapters/postgresql/books/migrations

# Postgres do books e do stats no Compose
BOOKS_DB_NAME=db_name
BOOKS_DB_PASS=change-me
BOOKS_DB_USER=db_user
STATS_DB_NAME=db_name
STATS_DB_PASS=change-me
STATS_DB_USER=db_user

# Serviços
PORT=:9090
STATS_PORT=:9091
NATS_URL=nats://localhost:4222
JWT_SECRET=change-the-secret   # a mesma chave nos dois serviços
```

<h3 id="starting">Iniciando</h3>

**Com Docker Compose (recomendado):**

```bash
docker compose up --build
```

| Serviço | Endereço |
|---|---|
| Gateway (use este) | http://localhost:8080 |
| books-service | http://localhost:9090 |
| stats-service | http://localhost:9091 |
| NATS (monitoramento) | http://localhost:8222 |
| Postgres books / stats | `localhost:5432` / `localhost:5433` |

Para conferir se os serviços subiram, chame `GET /health` em cada um.

**Sem Docker (desenvolvimento)**, com os Postgres e o NATS no ar (por exemplo, `docker compose up books-postgres stats-postgres nats`):

```bash
goose up                     # migrations do books (lê GOOSE_* do ambiente)
goose -dir ./internal/adapters/postgresql/stats/migrations postgres "$STATS_DBSTRING" up
go run ./cmd                 # books-service
go run ./cmd/stats           # stats-service (em outro terminal)
```

**Comandos úteis:**

```bash
go test ./...                # testes
go vet ./... && gofmt -l .   # checagens antes de commitar
sqlc generate                # regenera o código de acesso ao banco
docker compose down -v       # derruba tudo e apaga os volumes
```

<h2 id="routes">📍 Endpoints da API</h2>

Todas as respostas são em JSON, com campos em `snake_case`. Os erros vêm como `{"error": "mensagem"}`, em português.

As rotas de `/books` e `/stats` exigem o cabeçalho `Authorization: Bearer <token>` (o token vem do login). Sem token, ou com token inválido ou vencido → **401**. Cada usuário só vê e altera os próprios dados.

| Rota | Descrição |
|---|---|
| <kbd>POST /auth/register</kbd> | Cria uma conta, veja os [detalhes](#post-register-detail) |
| <kbd>POST /auth/login</kbd> | Entra e devolve o token JWT, veja os [detalhes](#post-login-detail) |
| <kbd>POST /books</kbd> | Cadastra um livro, veja os [detalhes](#post-books-detail) |
| <kbd>GET /books?status=</kbd> | Lista os livros, com filtro opcional por status, veja os [detalhes](#get-books-detail) |
| <kbd>GET /books/{id}</kbd> | Detalhe de um livro, veja os [detalhes](#get-book-detail) |
| <kbd>PATCH /books/{id}/progress</kbd> | Atualiza a página atual e publica o evento, veja os [detalhes](#patch-progress-detail) |
| <kbd>GET /stats/daily?days=</kbd> | Páginas lidas por dia, veja os [detalhes](#get-daily-detail) |
| <kbd>GET /stats/monthly?months=</kbd> | Livros concluídos por mês, veja os [detalhes](#get-monthly-detail) |
| <kbd>GET /stats/streak</kbd> | Sequência atual de dias lendo, veja os [detalhes](#get-streak-detail) |
| <kbd>GET /health</kbd> | Saúde do serviço (books e stats) |

<h3 id="post-register-detail">POST /auth/register</h3>

**REQUEST**
```json
{
  "email": "ana@exemplo.com",
  "password": "uma-senha-forte"
}
```

**RESPONSE** `201 Created`
```json
{
  "id": "5f0c2b1e-...",
  "email": "ana@exemplo.com",
  "created_at": "2026-11-21T14:30:00Z"
}
```

Erros: e-mail inválido ou senha fora de 8 a 72 caracteres → **400**; e-mail já cadastrado → **409**.

<h3 id="post-login-detail">POST /auth/login</h3>

**REQUEST**
```json
{
  "email": "ana@exemplo.com",
  "password": "uma-senha-forte"
}
```

**RESPONSE** `200 OK`
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

Erros: e-mail ou senha errados → **401**, com mensagem genérica (não diz qual dos dois errou).

<h3 id="post-books-detail">POST /books</h3>

**REQUEST** (`isbn`, `rating`, `notes` e `cover_url` são opcionais)
```json
{
  "title": "O Hobbit",
  "author": "J.R.R. Tolkien",
  "isbn": "9788595084742",
  "total_pages": 336,
  "rating": 5,
  "notes": "Presente de aniversário",
  "cover_url": "https://exemplo.com/capa.jpg",
  "owned": true
}
```

**RESPONSE** `201 Created`
```json
{
  "id": "b3f1c9d2-...",
  "user_id": "5f0c2b1e-...",
  "title": "O Hobbit",
  "author": "J.R.R. Tolkien",
  "isbn": "9788595084742",
  "total_pages": 336,
  "current_page": 0,
  "status": "wanna_read",
  "rating": 5,
  "notes": "Presente de aniversário",
  "cover_url": "https://exemplo.com/capa.jpg",
  "owned": true,
  "started_at": null,
  "finished_at": null,
  "created_at": "2026-11-21T14:30:00Z",
  "updated_at": "2026-11-21T14:30:00Z"
}
```

Erros: campos inválidos → **400**; ISBN repetido para o mesmo usuário → **409**.

<h3 id="get-books-detail">GET /books?status=</h3>

O filtro `status` é opcional e aceita `wanna_read`, `reading` ou `read`.

**RESPONSE** `200 OK`: uma lista de livros, no mesmo formato da resposta do `POST /books`.

<h3 id="get-book-detail">GET /books/{id}</h3>

**RESPONSE** `200 OK`: um livro, no mesmo formato da resposta do `POST /books`.

Erros: `id` inválido → **400**; livro não encontrado → **404**.

<h3 id="patch-progress-detail">PATCH /books/{id}/progress</h3>

Grava a atualização em `progress_updates`, ajusta o status do livro e publica o evento [`ReadingProgressUpdated`](#event).

**REQUEST**
```json
{
  "current_page": 120
}
```

**RESPONSE** `200 OK`: o livro atualizado, no mesmo formato da resposta do `POST /books`.

Erros: página fora do intervalo 0..`total_pages` → **400**; livro não encontrado → **404**.

<h3 id="get-daily-detail">GET /stats/daily?days=</h3>

Páginas lidas por dia nos últimos `days` dias. Aceita **7, 15 ou 30** (padrão **7**). Dias sem leitura não aparecem.

**RESPONSE** `200 OK`
```json
[
  { "day": "2026-11-20", "pages_read": 35, "updates": 2 },
  { "day": "2026-11-21", "pages_read": 20, "updates": 1 }
]
```

<h3 id="get-monthly-detail">GET /stats/monthly?months=</h3>

Livros concluídos por mês nos últimos `months` meses. Aceita **2, 3, 6 ou 12** (padrão **12**).

**RESPONSE** `200 OK`
```json
[
  { "month": "2026-10-01", "finished_count": 2 },
  { "month": "2026-11-01", "finished_count": 1 }
]
```

<h3 id="get-streak-detail">GET /stats/streak</h3>

**RESPONSE** `200 OK`
```json
{
  "current_streak": 5
}
```

Nos endpoints de estatísticas, um período fora da lista → **400** (ex.: `"Período inválido: Use 7, 15 ou 30"`). O "dia" é sempre o de **America/Sao_Paulo**.

<h2 id="event">📨 Evento ReadingProgressUpdated</h2>

Publicado pelo `books-service` no subject `books.progress.updated` **depois** que o progresso é gravado. Vai para o stream **`BOOKS`** do JetStream e é lido pelo `stats-service` por um **consumer durável**: se o stats estiver fora do ar, os eventos esperam no stream.

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

- **Sem duplicatas na publicação:** o `event_id` vai como `Nats-Msg-Id`, e o JetStream descarta publicações repetidas.
- **Consumer idempotente:** um evento repetido (mesmo `event_id`) é ignorado e não altera as estatísticas.
- **Uma transação por evento:** o evento e os resumos são gravados juntos, e só então a mensagem é confirmada (`Ack`).
- **Consistência eventual:** logo depois de um `PATCH`, as estatísticas podem mostrar o valor antigo por um instante.

<h2 id="roadmap">🗺️ Roadmap</h2>

| Período | Foco | |
|---|---|---|
| Semanas 1–5 | Rotas do books, Postgres, testes, config por env, Docker | ✅ |
| 16/11 | Gateway Nginx no Compose (`/books` + CORS) | ✅ |
| 17–19/11 | `stats-service` + `stats_db` + endpoints `/stats/*` | ✅ |
| 19–20/11 | NATS JetStream: publisher no books e consumer idempotente no stats | ✅ |
| 20–21/11 | Login com JWT (`users`, `/auth`, middleware nos dois serviços) | ✅ |
| 21/11 | App Flutter mínimo servido pelo gateway | 🚧 |
| 22/11 | Deploy sem Docker em VMs da AWS e do Azure | ⏳ |
| 23–26/11 | Deploy do Compose numa VM da AWS | ⏳ |
| 27–29/11 | Serviços em provedores diferentes, sem containers, via NATS | ⏳ |
| 30/11–03/12 | Os mesmos serviços com containers em múltiplos provedores | ⏳ |
| 05/12–10/01 | Telas restantes do app, `/healthz`, graceful shutdown, apresentação | ⏳ |

<h2 id="colab">🤝 Colaboradores</h2>

<table>
  <tr>
    <td align="center">
      <a href="https://github.com/DevDaleee">
        <img src="https://github.com/DevDaleee.png" width="100px;" alt="Foto de perfil de DevDaleee no GitHub"/><br>
        <sub>
          <b>DevDaleee</b>
        </sub>
      </a>
    </td>
  </tr>
</table>

<h2 id="contribute">📫 Contribuindo</h2>

Este é um projeto acadêmico e pessoal, mas sugestões são bem-vindas:

1. `git clone https://github.com/DevDaleee/my-personal-lib.git`
2. `git checkout -b feature/NOME`
3. Siga o padrão de commits do projeto: [Conventional Commits](https://www.conventionalcommits.org/pt-br/), com a mensagem em português no imperativo (ex.: `feat(stats): adiciona GET /stats/streak`)
4. Rode `go test ./...`, `go vet ./...` e `gofmt -l .` antes de abrir o Pull Request
5. Abra um Pull Request explicando o problema resolvido ou a funcionalidade criada

<h3>Documentações que podem ajudar</h3>

[📝 Como criar um Pull Request](https://www.atlassian.com/br/git/tutorials/making-a-pull-request)

[💾 Padrão de commits](https://gist.github.com/joshbuchea/6f47e86d2510bce28f8e7f42ae84c716)
