package main

import (
	"log"
	"net/http"
	"time"

	"accountapi/internal/entrypoint/httpapi"
	"accountapi/internal/infrastructure/repository/memrepo"
	"accountapi/internal/usecase"
)

func main() {
	port := httpapi.Env("PORT", "8080")
	seed := httpapi.Env("SEED_TEST_USER", "true")

	repo := memrepo.New()
	uc := &usecase.Usecase{Repo: repo}

	if seed == "true" {
		seedIfNeeded(uc)
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      httpapi.New(uc),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Printf("listening on :%s", port)
	log.Fatal(srv.ListenAndServe())
}

func seedIfNeeded(uc *usecase.Usecase) {
	if _, err := uc.GetUser("TaroYamada", "TaroYamada", "PaSSwd4TY"); err == nil {
		return
	}
	user, err := uc.SignUp("TaroYamada", "PaSSwd4TY")
	if err != nil {
		return
	}
	nn := strPtr("たろー")
	cm := strPtr("僕は元気です")
	_, _ = uc.UpdateUser(user.UserID, user.UserID, "PaSSwd4TY", nn, cm, false)
}

func strPtr(s string) *string { return &s }
