package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
)

func (s *server) lookupLongURL(ctx context.Context, code string) (string, error) {
	var longURL string

	err := s.replica.QueryRow(
		ctx,
		"SELECT long_url FROM links WHERE code = $1",
		code,
	).Scan(&longURL)

	if err == nil {
		return longURL, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("replica lookup failed for %q: %v", code, err)
	}

	// Replica missed the row or failed. Try the primary.
	err = s.db.QueryRow(
		ctx,
		"SELECT long_url FROM links WHERE code = $1",
		code,
	).Scan(&longURL)

	return longURL, err
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
	for range 5 {
		code, err = generateCode()
		if err != nil {
			log.Printf("generate short code: %v", err)
			http.Error(w, "failed to generate code", http.StatusInternalServerError)
			return
		}

		_, err = s.db.Exec(
			r.Context(),
			"INSERT INTO links (code, long_url) VALUES ($1, $2)",
			code,
			req.URL,
		)

		if err != nil {
			var pgErr *pgconn.PgError

			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				continue
			}

			log.Printf("create link: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		created = true
		break
	}

	if !created {
		log.Printf("create link: no unique code after 5 attempts")
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
	key := "link:" + code

	longURL, err := s.cache.Get(r.Context(), key).Result()

	if err == nil {
		w.Header().Set("X-Cache", "HIT")
	} else {
		w.Header().Set("X-Cache", "MISS")
		if !errors.Is(err, redis.Nil) {
			log.Printf("redis cache get %q: %v", code, err)
		}

		longURL, err = s.lookupLongURL(r.Context(), code)

		if errors.Is(err, pgx.ErrNoRows) {
			http.NotFound(w, r)
			return
		}

		if err != nil {
			log.Printf("database lookup %q: %v", code, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if cacheErr := s.cache.Set(
			r.Context(),
			key,
			longURL,
			24*time.Hour,
		).Err(); cacheErr != nil {
			log.Printf("redis cache set %q: %v", code, cacheErr)
		}
	}

	host, _ := os.Hostname()
	w.Header().Set("X-Served-By", host)

	http.Redirect(w, r, longURL, http.StatusFound)
}
