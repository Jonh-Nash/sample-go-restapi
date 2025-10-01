package httpapi

import (
	"encoding/json"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeAuthFailed(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="account-api"`)
	writeJSON(w, http.StatusUnauthorized, messageOnly{Message: "Authentication failed"})
}
