package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func logAuthError(r *http.Request, reason string, details map[string]any) {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		detailsJSON = []byte(`{"marshal_error":"failed to marshal auth details"}`)
	}

	log.Printf(
		"auth_error reason=%s method=%s path=%s remote_addr=%s user_agent=%q details=%s",
		reason,
		r.Method,
		r.URL.Path,
		clientIP(r),
		r.UserAgent(),
		string(detailsJSON),
	)
}

func clientIP(r *http.Request) string {
	if fwd := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}
