package kvdemo

import (
	"sync"

	"github.com/krisalay/distributed-systems-journal/distributedclock/hlc"
)

type Value struct {
	Data string
	TS   hlc.Timestamp
}

type Store struct {
	mu   sync.Mutex
	data map[string]Value
}

func NewStore() *Store {
	return &Store{data: make(map[string]Value)}
}

func (s *Store) Apply(key string, val Value) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.data[key]
	if !ok || hlc.DefinitelyAfter(val.TS, existing.TS) {
		s.data[key] = val
	}
}

func (s *Store) Data() map[string]Value {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := make(map[string]Value)
	for k, v := range s.data {
		copy[k] = v
	}
	return copy
}
