package filestore

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

type sourceRead struct {
	Read time.Time       `json:"read"`
	Kept json.RawMessage `json:"kept"`
}

type ReadsFile struct {
	path string
	mu   sync.Mutex
}

func NewReadsFile(path string) *ReadsFile {
	return &ReadsFile{path: path}
}

func (f *ReadsFile) Recall(name string, kept any) (time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	all, err := f.load()
	if err != nil {
		return time.Time{}, err
	}

	last, found := all[name]
	if !found {
		return time.Time{}, nil
	}

	if err := json.Unmarshal(last.Kept, kept); err != nil {
		return time.Time{}, err
	}

	return last.Read, nil
}

func (f *ReadsFile) Remember(name string, read time.Time, kept any) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	all, err := f.load()
	if err != nil {
		return err
	}

	data, err := json.Marshal(kept)
	if err != nil {
		return err
	}

	all[name] = sourceRead{Read: read, Kept: data}

	out, err := json.Marshal(all)
	if err != nil {
		return err
	}

	return writeAtomic(f.path, out)
}

func (f *ReadsFile) load() (map[string]sourceRead, error) {
	data, err := os.ReadFile(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]sourceRead{}, nil
	}

	if err != nil {
		return nil, err
	}

	all := map[string]sourceRead{}

	if err := json.Unmarshal(data, &all); err != nil {
		return nil, err
	}

	return all, nil
}
