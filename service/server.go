package main

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type server struct {
	db      *pgxpool.Pool
	replica *pgxpool.Pool
	cache   *redis.Client
	baseURL string
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /api/links", s.handleCreateLink)
	mux.HandleFunc("GET /{code}", s.handleRedirect)

	return mux
}
