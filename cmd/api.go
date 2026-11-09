package main

import (
	"log"
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

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("All good"))
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
