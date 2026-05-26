package handler

import (
	"net/http"
	"time"

	"mock-server-go/internal/auth"
)

// TokenHandler handles POST /oauth2/token using HTTP Basic Auth.
type TokenHandler struct {
	clientID     string
	clientSecret string
	tokens       *auth.TokenStore
}

func NewTokenHandler(clientID, clientSecret string, tokens *auth.TokenStore) *TokenHandler {
	return &TokenHandler{clientID: clientID, clientSecret: clientSecret, tokens: tokens}
}

func (h *TokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID, clientSecret, ok := r.BasicAuth()
	if !ok {
		logAuthError(r, "missing_or_invalid_basic_auth", map[string]any{
			"has_authorization_header": r.Header.Get("Authorization") != "",
		})
		w.Header().Set("WWW-Authenticate", `Basic realm="mock-server"`)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid basic auth"})
		return
	}

	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid form data"})
		return
	}

	grantType := r.FormValue("grant_type")
	if grantType != "client_credentials" ||
		clientID != h.clientID ||
		clientSecret != h.clientSecret {
		logAuthError(r, "invalid_client", map[string]any{
			"grant_type":            grantType,
			"client_id":             clientID,
			"client_id_match":       clientID == h.clientID,
			"client_secret_provided": clientSecret != "",
			"client_secret_match":   clientSecret == h.clientSecret,
		})
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_client"})
		return
	}

	const ttl = 3600 * time.Second
	token, err := h.tokens.Generate(ttl)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not generate token"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": token,
		"expires_in":   int(ttl.Seconds()),
	})
}
