package engine

import "sync"

type State struct {
	mu   sync.RWMutex
	data map[string]any
}

func NewState() *State {
	return &State{
		data: make(map[string]any),
	}
}

func (s *State) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

func (s *State) Get(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	return v, ok
}
