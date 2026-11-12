package filestore

import (
	"encoding/json"
	"errors"
	"os"
)

type JSONFile[T any] struct {
	path string
}

func NewJSONFile[T any](path string) *JSONFile[T] {
	return &JSONFile[T]{path: path}
}

func (f *JSONFile[T]) Load(missing T) (T, error) {
	data, err := os.ReadFile(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return missing, nil
	}

	if err != nil {
		return missing, err
	}

	var saved T
	if err := json.Unmarshal(data, &saved); err != nil {
		return missing, err
	}

	return saved, nil
}

func (f *JSONFile[T]) Save(v T) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	return writeAtomic(f.path, data)
}
