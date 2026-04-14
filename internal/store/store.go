package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Metadata struct {
	Color string `json:"color,omitempty"`
	Icon  string `json:"icon,omitempty"`
	Note  string `json:"note,omitempty"`
}

type Store struct {
	path string
	data map[string]Metadata
}

func New(repoRoot string) (*Store, error) {
	dir := filepath.Join(repoRoot, ".grove")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	s := &Store{
		path: filepath.Join(dir, "worktrees.json"),
		data: make(map[string]Metadata),
	}

	raw, err := os.ReadFile(s.path)
	if err == nil {
		json.Unmarshal(raw, &s.data)
	}

	return s, nil
}

func (s *Store) Get(name string) Metadata {
	return s.data[name]
}

func (s *Store) Set(name string, meta Metadata) error {
	if meta.Color == "" && meta.Icon == "" && meta.Note == "" {
		delete(s.data, name)
	} else {
		s.data[name] = meta
	}
	return s.save()
}

func (s *Store) Delete(name string) error {
	delete(s.data, name)
	return s.save()
}

func (s *Store) save() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0644)
}
