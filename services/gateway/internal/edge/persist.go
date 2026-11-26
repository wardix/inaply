package edge

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/inaply/gateway/internal/models"
)

// FileStore persists gateway edge state atomically on local disk so reboots or
// cloud internet dropouts never interrupt hotel room sessions or cached promos.
type FileStore struct {
	mu   sync.Mutex
	path string
}

func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

func (f *FileStore) Save(snap models.EdgeSnapshot) error {
	if f.path == "" {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".edge-state-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, f.path)
}

func (f *FileStore) Load() (*models.EdgeSnapshot, error) {
	if f.path == "" {
		return nil, os.ErrNotExist
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	raw, err := os.ReadFile(f.path)
	if err != nil {
		return nil, err
	}
	var snap models.EdgeSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		return nil, err
	}
	if snap.Rooms == nil || snap.Receivers == nil {
		return nil, errors.New("corrupt edge snapshot: missing rooms or receivers")
	}
	if snap.Sessions == nil {
		snap.Sessions = make(map[string]*models.GuestSession)
	}
	return &snap, nil
}
