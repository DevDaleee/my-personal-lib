package main

import (
	"log"
	_ "my-personal-lib/cmd/docs" // registra o spec gerado pelo swag (o import roda o init() do pacote)
	repo "my-personal-lib/internal/adapters/postgresql/books/sqlc"
	"my-personal-lib/internal/auth"
	"my-personal-lib/internal/books"
	messages "my-personal-lib/internal/error"
	"my-personal-lib/internal/json"
	"my-personal-lib/internal/userctx"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(middleware.Timeout((60 * time.Second)))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		json.WriteError(w, http.StatusNotFound, messages.ErrNotFound.Error())
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		json.WriteError(w, http.StatusMethodNotAllowed, messages.ErrMethodNotAllowed.Error())
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("All good"))
	})

	// Documentação Swagger (gerada pelo "swag init" em cmd/docs). Fica fora do
	// middleware JWT, como o /health. Abrir em /swagger/index.html.
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	bookService := books.NewService(repo.New(app.db), app.db, app.js)
	bookHandler := books.NewHandler(bookService)
	r.Route("/books", func(r chi.Router) {
		r.Use(userctx.Middleware(app.config.jwt))
		r.Post("/", bookHandler.CreateBook)
		r.Get("/{id}", bookHandler.GetBookByID)
		r.Get("/", bookHandler.GetBookByStatus)
		r.Patch("/{id}/progress", bookHandler.UpdateBookProgress)
	})

	authService := auth.NewService(repo.New(app.db), app.config.jwt)
	authHandler := auth.NewHandler(authService)
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
	})
	r.Route("/users", func(r chi.Router) {
		r.Use(userctx.Middleware(app.config.jwt))
		r.Get("/me", authHandler.GetMe)
	})

	return r
}

func (app *application) run(h http.Handler) error {
	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      h,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	log.Println("Server has started at addr", app.config.addr)
	return srv.ListenAndServe()
}

type application struct {
	config config
	db     *pgxpool.Pool
	js     jetstream.JetStream
}
type config struct {
	addr string
	db   dbConfig
	jwt  []byte
}
type dbConfig struct {
	dsn string
}
