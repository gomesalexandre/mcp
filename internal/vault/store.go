package vault

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

// Info holds the vault key material for a session.
type Info struct {
	ECDSAPublicKey string
	EdDSAPublicKey string
	ChainCode      string
}

// Calldata holds encoded calldata produced by abi_encode or an upstream
// proxy. Referenced by a short calldata_id that the LLM passes between tools.
type Calldata struct {
	To   string // destination contract address
	Data string // hex-encoded calldata (0x-prefixed)
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
func (s *Store) StoreCalldata(cd Calldata) string {
	id := generateCalldataID()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calldata[id] = cd
	return id
}

// GetCalldata retrieves stored calldata by ID.
func (s *Store) GetCalldata(id string) (Calldata, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cd, ok := s.calldata[id]
	return cd, ok
}

func generateCalldataID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return "cd_" + hex.EncodeToString(b)
}
