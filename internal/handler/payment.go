package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"mock-server-go/internal/auth"
	"mock-server-go/internal/store"
)

// PaymentHandler handles payment-related routes.
type PaymentHandler struct {
	tokens        *auth.TokenStore
	notifications *store.NotificationStore
}

func NewPaymentHandler(tokens *auth.TokenStore, notifications *store.NotificationStore) *PaymentHandler {
	return &PaymentHandler{tokens: tokens, notifications: notifications}
}

// CompletePayment handles POST /ws/payment/completePayment/v1.
func (h *PaymentHandler) CompletePayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		logAuthError(r, "missing_bearer_token", map[string]any{
			"authorization_header_present": authHeader != "",
			"authorization_scheme":         headerScheme(authHeader),
		})
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if valid, reason := h.tokens.Validate(token); !valid {
		logAuthError(r, "invalid_bearer_token", map[string]any{
			"validation_reason": reason,
			"token_len":         len(token),
			"token_prefix":      tokenPrefix(token),
		})
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
		return
	}

	var body struct {
		ReferenceNumber     string `json:"reference_number"`
		TransactionDateTime string `json:"transaction_date_time"`
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "could not read body"})
		return
	}
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if body.ReferenceNumber == "" || body.TransactionDateTime == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "reference_number and transaction_date_time are required"})
		return
	}

	h.notifications.Add(store.PaymentNotification{
		ReferenceNumber:     body.ReferenceNumber,
		TransactionDateTime: body.TransactionDateTime,
		ReceivedAt:          time.Now(),
		RawPayload:          json.RawMessage(raw),
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"status": map[string]any{
			"fault_code": 0,
			"message":    "success",
		},
	})
}

func headerScheme(authHeader string) string {
	authHeader = strings.TrimSpace(authHeader)
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	return strings.ToLower(parts[0])
}

func tokenPrefix(token string) string {
	const n = 8
	if len(token) <= n {
		return token
	}
	return token[:n]
}

// ListNotifications handles GET /ws/payment/notifications.
func (h *PaymentHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	items := h.notifications.List()
	writeJSON(w, http.StatusOK, map[string]any{
		"count": len(items),
		"items": items,
	})
}
