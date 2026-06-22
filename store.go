package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
)

// Store es un almacén en memoria thread-safe de pagos.
// Indexa por ID interno y por QRId del BNB para búsquedas rápidas desde el webhook.
type Store struct {
	mu       sync.RWMutex
	byID     map[string]*Payment // clave: nuestro UUID
	byQRId   map[string]*Payment // clave: QRId del BNB
}

func NewStore() *Store {
	return &Store{
		byID:   make(map[string]*Payment),
		byQRId: make(map[string]*Payment),
	}
}

func (s *Store) Save(p *Payment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[p.ID] = p
	s.byQRId[p.QRId] = p
}

func (s *Store) GetByID(id string) (*Payment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.byID[id]
	return p, ok
}

func (s *Store) GetByQRId(qrId string) (*Payment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.byQRId[qrId]
	return p, ok
}

func (s *Store) Update(p *Payment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[p.ID]; !ok {
		return fmt.Errorf("payment %s not found", p.ID)
	}
	s.byID[p.ID] = p
	s.byQRId[p.QRId] = p
	return nil
}

// newID genera un ID hexadecimal de 16 bytes para los pagos.
func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
