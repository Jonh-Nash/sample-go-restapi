package main

import (
	"log"
	"net/http"
	"time"

	"accountapi/internal/entrypoint/httpapi"
)

func main() {
	port := httpapi.Env("PORT", "8080")

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      httpapi.New(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Printf("listening on :%s", port)
	log.Fatal(srv.ListenAndServe())
}
