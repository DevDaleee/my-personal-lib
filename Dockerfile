# Estágio 1: compila o books-service com o Go completo.
FROM golang:1.27-alpine AS builder
WORKDIR /src

# Dependências primeiro: esta camada fica em cache enquanto go.mod/go.sum não mudarem.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=0 gera um executável estático, que roda no alpine sem dependências.
RUN CGO_ENABLED=0 go build -o /out/books-service ./cmd

# Estágio 2: compila a CLI do goose, na mesma versão usada fora do Docker.
FROM golang:1.27-alpine AS goose-builder
RUN CGO_ENABLED=0 go install github.com/pressly/goose/v3/cmd/goose@v3.29.0

# Estágio 3 (alvo "migrate"): roda as migrations uma vez e termina.
# Usado pelo serviço migrate do docker-compose.yaml (target: migrate).
FROM alpine:3.22 AS migrate
RUN adduser -D -H app
COPY --from=goose-builder /go/bin/goose /usr/local/bin/goose
# --chown: os .sql vêm com permissão só para o dono; sem isto o usuário app não consegue lê-los.
COPY --chown=app:app internal/adapters/postgresql/books/migrations /migrations/books
COPY --chown=app:app internal/adapters/postgresql/stats/migrations /migrations/stats
USER app
# O goose lê GOOSE_DRIVER, GOOSE_DBSTRING e GOOSE_MIGRATION_DIR do ambiente.
CMD ["goose", "up"]

# Estágio 4 (final): imagem mínima do books-service, só com o executável.
# Fica por último porque é o alvo padrão de um "docker build" sem --target.
FROM alpine:3.22

# Usuário comum: o programa não roda como root dentro do container.
RUN adduser -D -H app

COPY --from=builder /out/books-service /usr/local/bin/books-service

USER app
EXPOSE 9090

# Exec form: o programa é o processo principal e recebe os sinais de desligamento.
CMD ["books-service"]
