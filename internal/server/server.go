package server

import (
	"log"
	"net/http"
	"os"

	"mock-server-go/internal/auth"
	"mock-server-go/internal/handler"
	"mock-server-go/internal/store"
)

type Config struct {
	Port         string
	ClientID     string
	ClientSecret string
}

func ConfigFromEnv() Config {
	return Config{
		Port:         getEnv("PORT", "8080"),
		ClientID:     getEnv("NSW_CLIENT_ID", "mock-client-id"),
		ClientSecret: getEnv("NSW_CLIENT_SECRET", "mock-client-secret"),
	}
}

func Run(cfg Config) error {
	tokens := auth.NewTokenStore()
	notifications := store.NewNotificationStore()

	tokenHandler := handler.NewTokenHandler(cfg.ClientID, cfg.ClientSecret, tokens)
	paymentHandler := handler.NewPaymentHandler(tokens, notifications)

	mux := http.NewServeMux()
	mux.Handle("/oauth2/token", tokenHandler)
	mux.HandleFunc("/ws/payment/completePayment/v1", paymentHandler.CompletePayment)
	mux.HandleFunc("/ws/payment/notifications", paymentHandler.ListNotifications)

	log.Printf("Mock server starting on :%s", cfg.Port)
	return http.ListenAndServe(":"+cfg.Port, mux)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
