package main

import (
	"os"
	"strings"
)

type appConfig struct {
	Port string
}

func loadConfig() (appConfig, error) {
	cfg := appConfig{
		Port: "8080",
	}

	if v, ok := lookupNonEmptyEnv("PORT"); ok {
		cfg.Port = v
	}

	return cfg, nil
}

func lookupNonEmptyEnv(key string) (string, bool) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}
