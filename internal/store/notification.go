package store

import (
	"sync"
	"time"
)

type PaymentNotification struct {
	ReferenceNumber     string    `json:"reference_number"`
	TransactionDateTime string    `json:"transaction_date_time"`
	ReceivedAt          time.Time `json:"received_at"`
}

// NotificationStore is a thread-safe in-memory store for payment notifications.
type NotificationStore struct {
	mu    sync.RWMutex
	items []PaymentNotification
}

func NewNotificationStore() *NotificationStore {
	return &NotificationStore{}
}

func (s *NotificationStore) Add(n PaymentNotification) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, n)
}

func (s *NotificationStore) List() []PaymentNotification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]PaymentNotification, len(s.items))
	copy(result, s.items)
	return result
}
