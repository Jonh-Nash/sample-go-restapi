package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	defaultBaseURL   = "http://localhost:8080"
	defaultUserID    = "TaroYamada"
	defaultPassword  = "PaSSwd4TY"
	defaultNickname  = "たろー"
	defaultComment   = "僕は元気です"
	defaultWaitLimit = 30 * time.Second
)

type config struct {
	BaseURL   string
	UserID    string
	Password  string
	Nickname  string
	Comment   string
	WaitLimit time.Duration
}

func main() {
	cfg := loadConfig()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.WaitLimit)
	defer cancel()

	if err := waitForServer(ctx, cfg.BaseURL); err != nil {
		log.Fatalf("server is not ready: %v", err)
	}

	if err := ensureTestUser(ctx, cfg); err != nil {
		log.Fatalf("failed to seed test user: %v", err)
	}

	log.Printf("seed completed for user %q", cfg.UserID)
}

func loadConfig() config {
	cfg := config{
		BaseURL:   defaultBaseURL,
		UserID:    defaultUserID,
		Password:  defaultPassword,
		Nickname:  defaultNickname,
		Comment:   defaultComment,
		WaitLimit: defaultWaitLimit,
	}

	if v, ok := lookupEnv("API_BASE_URL"); ok {
		cfg.BaseURL = strings.TrimRight(v, "/")
	}
	if v, ok := lookupEnv("SEED_USER_ID"); ok {
		cfg.UserID = v
	}
	if v, ok := lookupEnv("SEED_PASSWORD"); ok {
		cfg.Password = v
	}
	if v, ok := lookupEnv("SEED_NICKNAME"); ok && utf8.ValidString(v) {
		cfg.Nickname = v
	}
	if v, ok := lookupEnv("SEED_COMMENT"); ok && utf8.ValidString(v) {
		cfg.Comment = v
	}
	if v, ok := lookupEnv("SEED_WAIT_LIMIT"); ok {
		d, err := time.ParseDuration(v)
		if err == nil && d > 0 {
			cfg.WaitLimit = d
		}
	}

	return cfg
}

func lookupEnv(key string) (string, bool) {
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

func waitForServer(ctx context.Context, baseURL string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("%s/healthz", strings.TrimRight(baseURL, "/"))
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

var errUserNotFound = errors.New("user not found")

func ensureTestUser(ctx context.Context, cfg config) error {
	client := &http.Client{Timeout: 5 * time.Second}

	exists, err := userExists(ctx, client, cfg)
	if err != nil {
		if !errors.Is(err, errUserNotFound) {
			return fmt.Errorf("check user: %w", err)
		}
	}

	if !exists {
		if err := signUpUser(ctx, client, cfg); err != nil {
			return fmt.Errorf("signup: %w", err)
		}
	}

	if err := updateProfile(ctx, client, cfg); err != nil {
		return fmt.Errorf("update profile: %w", err)
	}

	return nil
}

func userExists(ctx context.Context, client *http.Client, cfg config) (bool, error) {
	endpoint := fmt.Sprintf("%s/users/%s", strings.TrimRight(cfg.BaseURL, "/"), url.PathEscape(cfg.UserID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}
	req.SetBasicAuth(cfg.UserID, cfg.Password)

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, errUserNotFound
	case http.StatusUnauthorized:
		// The API returns 401 when the account is absent, so treat it as "not found" to allow initial seeding.
		return false, errUserNotFound
	default:
		return false, fmt.Errorf("unexpected status %d while checking user", resp.StatusCode)
	}
}

func signUpUser(ctx context.Context, client *http.Client, cfg config) error {
	endpoint := fmt.Sprintf("%s/signup", strings.TrimRight(cfg.BaseURL, "/"))
	body := struct {
		UserID   string `json:"user_id"`
		Password string `json:"password"`
	}{cfg.UserID, cfg.Password}

	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode == http.StatusBadRequest {
		var failure struct {
			Message string `json:"message"`
			Cause   string `json:"cause"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&failure); err != nil {
			return fmt.Errorf("unexpected signup failure with status 400")
		}
		if strings.EqualFold(failure.Cause, "Already same user_id is used") {
			return nil
		}
		return fmt.Errorf("signup failed: %s (%s)", failure.Message, failure.Cause)
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("signup failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
}

func updateProfile(ctx context.Context, client *http.Client, cfg config) error {
	endpoint := fmt.Sprintf("%s/users/%s", strings.TrimRight(cfg.BaseURL, "/"), url.PathEscape(cfg.UserID))
	body := struct {
		Nickname string `json:"nickname"`
		Comment  string `json:"comment"`
	}{cfg.Nickname, cfg.Comment}

	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(cfg.UserID, cfg.Password)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return errors.New("authentication failed when updating user")
	case http.StatusForbidden:
		return errors.New("forbidden from updating user")
	default:
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("update failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}
}
