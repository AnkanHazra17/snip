package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	links   = make(map[string]string)
	linksMu sync.RWMutex
)

type server struct {
	db      *pgxpool.Pool
	baseURL string
}

func (s *server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func (s *server) handleCreateLink(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	var req createLinkReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError

		if errors.As(err, &maxBytesErr) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}

		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	parsedURL, err := url.Parse(req.URL)

	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		http.Error(w, "invalid URL", http.StatusBadRequest)
		return
	}

	var code string
	created := false
	for attempt := 0; attempt < 5; attempt++ {
		code, err = generateCode()
		if err != nil {
			http.Error(w, "failed to generate code", http.StatusInternalServerError)
			return
		}

		linksMu.Lock()

		if _, exists := links[code]; exists {
			linksMu.Unlock()
			continue
		}

		links[code] = req.URL
		linksMu.Unlock()
		created = true
		break
	}

	if !created {
		http.Error(w, "failed to generate unique code", http.StatusInternalServerError)
		return
	}

	resp := createLinkResp{
		Code:     code,
		ShortURL: s.baseURL + "/" + code,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (s *server) handleRedirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	var longURL string

	err := s.db.QueryRow(
		r.Context(),
		"SELECT long_url FROM links WHERE code = $1",
		code,
	).Scan(&longURL)

	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}

	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, longURL, http.StatusFound)
}

type createLinkReq struct {
	URL string `json:"url"`
}

type createLinkResp struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateCode() (string, error) {
	code := make([]byte, 6)

	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(62))
		if err != nil {
			return "", err
		}

		code[i] = alphabet[n.Int64()]
	}

	return string(code), nil
}

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("failed to create database pool: %v", err)
	}

	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	log.Println("connected to database")

	mux := http.NewServeMux()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:" + port
	}

	baseURL = strings.TrimRight(baseURL, "/")

	s := &server{
		db:      pool,
		baseURL: baseURL,
	}

	mux.HandleFunc("GET /healthz", s.handleHealthz)

	mux.HandleFunc("POST /api/links", s.handleCreateLink)

	mux.HandleFunc("GET /{code}", s.handleRedirect)

	log.Println("listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
