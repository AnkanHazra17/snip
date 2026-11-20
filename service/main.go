package main

import (
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	pool, err := newPostgresPool("DATABASE_URL")
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	replicaPool, err := newPostgresPool("DATABASE_REPLICA_URL")
	if err != nil {
		log.Fatal(err)
	}
	defer replicaPool.Close()

	rdb := newRedisClient()
	defer rdb.Close()

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
		replica: replicaPool,
		cache:   rdb,
		baseURL: baseURL,
	}

	log.Println("listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, s.routes()))
}
