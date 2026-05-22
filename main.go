package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type tokenEntry struct {
	token     string
	expiresAt time.Time
}

type paymentNotification struct {
	ReferenceNumber     string    `json:"reference_number"`
	TransactionDateTime string    `json:"transaction_date_time"`
	ReceivedAt          time.Time `json:"received_at"`
}

var (
	tokenStore   = make(map[string]tokenEntry)
	tokenStoreMu sync.RWMutex

	notifications   []paymentNotification
	notificationsMu sync.RWMutex
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func storeToken(token string, ttl time.Duration) {
	tokenStoreMu.Lock()
	defer tokenStoreMu.Unlock()
	tokenStore[token] = tokenEntry{token: token, expiresAt: time.Now().Add(ttl)}
}

func isValidToken(token string) bool {
	tokenStoreMu.RLock()
	defer tokenStoreMu.RUnlock()
	entry, ok := tokenStore[token]
	return ok && time.Now().Before(entry.expiresAt)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

// POST /oauth2/token
func handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	clientID, clientSecret, ok := r.BasicAuth()
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="mock-server"`)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid basic auth"})
		return
	}

	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid form data"})
		return
	}

	grantType := r.FormValue("grant_type")

	expectedClientID := getEnv("NSW_CLIENT_ID", "mock-client-id")
	expectedClientSecret := getEnv("NSW_CLIENT_SECRET", "mock-client-secret")

	if grantType != "client_credentials" || clientID != expectedClientID || clientSecret != expectedClientSecret {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_client"})
		return
	}

	const ttl = 3600 * time.Second
	token, err := generateToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not generate token"})
		return
	}
	storeToken(token, ttl)

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": token,
		"expires_in":   int(ttl.Seconds()),
	})
}

// POST /ws/payment/completePayment/v1
func handleCompletePayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if !isValidToken(token) {
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

	notificationsMu.Lock()
	notifications = append(notifications, paymentNotification{
		ReferenceNumber:     body.ReferenceNumber,
		TransactionDateTime: body.TransactionDateTime,
		ReceivedAt:          time.Now(),
	})
	notificationsMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"status": map[string]any{
			"fault_code": 0,
			"message":    "success",
		},
	})
}

// GET /ws/payment/notifications
func handleListNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	notificationsMu.RLock()
	result := make([]paymentNotification, len(notifications))
	copy(result, notifications)
	notificationsMu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"count": len(result),
		"items": result,
	})
}

func main() {
	port := getEnv("PORT", "8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/token", handleToken)
	mux.HandleFunc("/ws/payment/completePayment/v1", handleCompletePayment)
	mux.HandleFunc("/ws/payment/notifications", handleListNotifications)

	log.Printf("Mock server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
