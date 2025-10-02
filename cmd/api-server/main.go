package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"accountapi/internal/entrypoint/rest"
)

func main() {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	handler := rest.New()
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Printf("listening on :%s", port)
	log.Fatal(srv.ListenAndServe())
}
