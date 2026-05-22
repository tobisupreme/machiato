package handler

import (
	"encoding/json"
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
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
		return
	}
	if !h.tokens.IsValid(strings.TrimPrefix(authHeader, "Bearer ")) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
		return
	}

	var body struct {
		ReferenceNumber     string `json:"reference_number"`
		TransactionDateTime string `json:"transaction_date_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
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
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"status": map[string]any{
			"fault_code": 0,
			"message":    "success",
		},
	})
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
