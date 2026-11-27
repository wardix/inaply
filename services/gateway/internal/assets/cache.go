package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Cache manages local LAN media assets and OTA APKs with SHA-256 verification.
type Cache struct {
	mu     sync.RWMutex
	dir    string
	client *http.Client
}

func NewCache(dir string, client *http.Client) *Cache {
	if client == nil {
		client = http.DefaultClient
	}
	_ = os.MkdirAll(dir, 0o755)
	return &Cache{
		dir:    dir,
		client: client,
	}
}

func (c *Cache) Dir() string {
	return c.dir
}

func validateFilename(name string) error {
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("invalid asset filename %q", name)
	}
	return nil
}

// Ensure downloads sourceURL, verifies its SHA-256 digest, and atomically stores it as filename.
func (c *Cache) Ensure(filename, sourceURL, expectedSHA256 string) error {
	if err := validateFilename(filename); err != nil {
		return err
	}
	if expectedSHA256 == "" {
		return errors.New("expected SHA-256 is required")
	}

	resp, err := c.client.Get(sourceURL)
	if err != nil {
		return fmt.Errorf("download %s: %w", sourceURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: unexpected status %d", sourceURL, resp.StatusCode)
	}

	tmp, err := os.CreateTemp(c.dir, filename+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		_ = tmp.Close()
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, hasher), resp.Body); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	gotHash := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(gotHash, expectedSHA256) {
		return fmt.Errorf("sha256 mismatch for %s: got %s want %s", filename, gotHash, expectedSHA256)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if err := os.Rename(tmpPath, filepath.Join(c.dir, filename)); err != nil {
		return fmt.Errorf("atomic rename: %w", err)
	}
	cleanup = false
	return nil
}

// Handler serves `/assets/<filename>` with HTTP Byte-Range support.
func (c *Cache) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/assets/")
		if err := validateFilename(name); err != nil {
			http.NotFound(w, r)
			return
		}

		c.mu.RLock()
		fullPath := filepath.Join(c.dir, name)
		c.mu.RUnlock()

		http.ServeFile(w, r, fullPath)
	})
}

// GC removes cached files not present in keepFilenames.
func (c *Cache) GC(keepFilenames []string) (int, error) {
	keep := make(map[string]struct{}, len(keepFilenames))
	for _, name := range keepFilenames {
		keep[name] = struct{}{}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return 0, err
	}

	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if _, ok := keep[entry.Name()]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(c.dir, entry.Name())); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}
