package filestore

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type state struct {
	Seen   map[string]time.Time `json:"seen"`
	Booted map[string]bool      `json:"booted"`
}

type SeenStore struct {
	mu    sync.Mutex
	path  string
	dirty bool
	state state
}

func Open(path string) (*SeenStore, error) {
	s := &SeenStore{path: path}
	s.ensureMaps()

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &s.state); err != nil {
		return nil, err
	}

	s.ensureMaps()

	return s, nil
}

func (s *SeenStore) MarkNew(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.state.Seen[id]; ok {
		return false
	}

	s.state.Seen[id] = time.Now().UTC()
	s.dirty = true

	return true
}

func (s *SeenStore) IsSeeded(source string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.state.Booted[source]
}

func (s *SeenStore) MarkSeeded(source string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state.Booted[source] {
		return
	}

	s.state.Booted[source] = true
	s.dirty = true
}

func (s *SeenStore) Prune(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)

	for id, at := range s.state.Seen {
		if at.Before(cutoff) {
			delete(s.state.Seen, id)
			s.dirty = true
		}
	}
}

func (s *SeenStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.dirty {
		return nil
	}

	if err := s.write(); err != nil {
		return err
	}

	s.dirty = false

	return nil
}

func (s *SeenStore) write() error {
	data, err := json.Marshal(s.state)
	if err != nil {
		return err
	}

	return writeAtomic(s.path, data)
}

func (s *SeenStore) ensureMaps() {
	if s.state.Seen == nil {
		s.state.Seen = map[string]time.Time{}
	}

	if s.state.Booted == nil {
		s.state.Booted = map[string]bool{}
	}
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tmp := path + ".tmp"

	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, path)
}
