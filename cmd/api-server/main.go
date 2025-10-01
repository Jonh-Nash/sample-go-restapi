package main

import (
	"log"
	"net/http"
	"time"

	"accountapi/internal/entrypoint/httpapi"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	handler := httpapi.New()

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Printf("listening on :%s", cfg.Port)
	log.Fatal(srv.ListenAndServe())
}
