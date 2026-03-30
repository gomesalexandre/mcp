package vault

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const calldataTTL = 10 * time.Minute

// Info holds the vault key material for a session.
type Info struct {
	ECDSAPublicKey string
	EdDSAPublicKey string
	ChainCode      string
}

// Calldata holds encoded calldata produced by abi_encode or an upstream
// proxy. Referenced by a short calldata_id that the LLM passes between tools.
type Calldata struct {
	To        string // destination contract address
	Data      string // hex-encoded calldata (0x-prefixed)
	createdAt time.Time
}

// Store is a concurrency-safe state store for vault info and calldata.
type Store struct {
	mu       sync.RWMutex
	vaults   map[string]Info
	calldata map[string]Calldata // keyed by calldata_id
}

func NewStore() *Store {
	return &Store{
		vaults:   make(map[string]Info),
		calldata: make(map[string]Calldata),
	}
}

func (s *Store) Set(sessionID string, info Info) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vaults[sessionID] = info
}

func (s *Store) Get(sessionID string) (Info, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	info, ok := s.vaults[sessionID]
	return info, ok
}

func (s *Store) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.vaults, sessionID)
}

// StoreCalldata saves calldata under a random ID and returns the ID.
// Expired entries are lazily evicted on each call.
func (s *Store) StoreCalldata(cd Calldata) string {
	id := generateCalldataID()
	cd.createdAt = time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, v := range s.calldata {
		if now.Sub(v.createdAt) > calldataTTL {
			delete(s.calldata, k)
		}
	}
	s.calldata[id] = cd
	return id
}

// GetCalldata retrieves stored calldata by ID. Returns false if expired or missing.
func (s *Store) GetCalldata(id string) (Calldata, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cd, ok := s.calldata[id]
	if !ok || time.Since(cd.createdAt) > calldataTTL {
		return Calldata{}, false
	}
	return cd, ok
}

func generateCalldataID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return "cd_" + hex.EncodeToString(b)
}
