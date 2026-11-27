package ops

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
)

// HealthcheckFunc verifies the newly applied binary is healthy; returning an error triggers rollback.
type HealthcheckFunc func() error

// SelfUpdater downloads gateway binary updates, verifies SHA-256, and rolls back on healthcheck failure.
type SelfUpdater struct {
	binPath     string
	client      *http.Client
	healthcheck HealthcheckFunc
}

func NewSelfUpdater(binPath string, client *http.Client, healthcheck HealthcheckFunc) *SelfUpdater {
	if client == nil {
		client = http.DefaultClient
	}
	return &SelfUpdater{
		binPath:     binPath,
		client:      client,
		healthcheck: healthcheck,
	}
}

func (u *SelfUpdater) Apply(sourceURL, expectedSHA256 string) error {
	if expectedSHA256 == "" {
		return errors.New("expected SHA-256 is required")
	}

	resp, err := u.client.Get(sourceURL)
	if err != nil {
		return fmt.Errorf("download update: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download update: unexpected status %d", resp.StatusCode)
	}

	dir := filepath.Dir(u.binPath)
	tmp, err := os.CreateTemp(dir, "gateway-update-*")
	if err != nil {
		return fmt.Errorf("create temp binary: %w", err)
	}
	tmpPath := tmp.Name()
	cleanupTmp := true
	defer func() {
		_ = tmp.Close()
		if cleanupTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, hasher), resp.Body); err != nil {
		return fmt.Errorf("write temp binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp binary: %w", err)
	}

	gotHash := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(gotHash, expectedSHA256) {
		return fmt.Errorf("checksum mismatch: got %s want %s", gotHash, expectedSHA256)
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return fmt.Errorf("chmod temp binary: %w", err)
	}

	backupPath := u.binPath + ".bak"
	if err := os.Rename(u.binPath, backupPath); err != nil {
		return fmt.Errorf("backup current binary: %w", err)
	}
	if err := os.Rename(tmpPath, u.binPath); err != nil {
		_ = os.Rename(backupPath, u.binPath)
		return fmt.Errorf("install new binary: %w", err)
	}
	cleanupTmp = false

	if u.healthcheck != nil {
		if err := u.healthcheck(); err != nil {
			_ = os.Rename(backupPath, u.binPath)
			return fmt.Errorf("healthcheck failed, rolled back: %w", err)
		}
	}
	_ = os.Remove(backupPath)
	return nil
}
