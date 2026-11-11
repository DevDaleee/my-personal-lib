package main

import (
	"log"
	repo "my-personal-lib/internal/adapters/postgresql/sqlc"
	"my-personal-lib/internal/books"
	messages "my-personal-lib/internal/error"
	"my-personal-lib/internal/json"
	"my-personal-lib/internal/userctx"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
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

	bookService := books.NewSerice(repo.New(app.db), app.db)
	bookHandler := books.NewHandler(bookService)
	r.Route("/books", func(r chi.Router) {
		r.Use(userctx.Middleware)
		r.Post("/", bookHandler.CreateBook)
		r.Get("/{id}", bookHandler.GetBook)
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
}
type config struct {
	addr string
	db   dbConfig
}
type dbConfig struct {
	dsn string
}
