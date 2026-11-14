package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"sync"
)

var (
	links   = make(map[string]string)
	linksMu sync.RWMutex
)

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
	mux := http.NewServeMux()

	PORT := os.Getenv("PORT")
	if PORT == "" {
		PORT = "8080"
	}
	baseURL := "http://localhost:" + PORT

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("POST /api/links", func(w http.ResponseWriter, r *http.Request) {
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
			ShortURL: baseURL + "/" + code,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("GET /{code}", func(w http.ResponseWriter, r *http.Request) {
		code := r.PathValue("code")

		linksMu.RLock()
		originalURL, ok := links[code]
		linksMu.RUnlock()

		if !ok {
			http.NotFound(w, r)
			return
		}

		http.Redirect(w, r, originalURL, http.StatusFound)
	})
	log.Println("listening on :" + PORT)
	log.Fatal(http.ListenAndServe(":"+PORT, mux))
}
