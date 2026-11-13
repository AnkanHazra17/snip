package main

import (
	"crypto/rand"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"net/url"
)

var links = make(map[string]string)

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

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("POST /api/links", func(w http.ResponseWriter, r *http.Request) {
		var req createLinkReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		parsedURL, err := url.Parse(req.URL)

		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
			http.Error(w, "invalid URL", http.StatusBadRequest)
			return
		}

		var code string
		for {
			code, err = generateCode()
			if err != nil {
				http.Error(w, "failed to generate code", http.StatusInternalServerError)
				return
			}

			if _, exists := links[code]; !exists {
				break
			}
		}

		links[code] = req.URL
		
		resp := createLinkResp{
			Code:     code,
			ShortURL: "http://localhost:8080/" + code,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	})
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
