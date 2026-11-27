package ops

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/inaply/gateway/internal/firewall"
	"github.com/inaply/gateway/internal/session"
)

func TestWatchdog_DetectsOfflineReceiver(t *testing.T) {
	store := session.NewStore(firewall.NewMemoryDriver(), nil)
	store.SeedDemoProperty()

	reachable := map[string]bool{
		"10.20.3.101": true,
		"10.20.3.102": true,
		"10.20.5.101": true,
	}
	wd := NewWatchdog(store, 2, func(ip string) bool {
		return reachable[ip]
	})

	// Fail 10.20.3.101 once -> still below threshold (2)
	reachable["10.20.3.101"] = false
	wd.ProbeOnce()
	st1, _ := store.TVState("tv-301-main")
	if st1.DeviceStatus == "offline" {
		t.Fatalf("expected tv-301-main not offline after 1 failure")
	}

	// Fail 10.20.3.101 twice -> marked offline
	wd.ProbeOnce()
	st2, _ := store.TVState("tv-301-main")
	if st2.DeviceStatus != "offline" {
		t.Fatalf("expected tv-301-main offline after 2 failures, got %q", st2.DeviceStatus)
	}

	// Recover -> marked idle
	reachable["10.20.3.101"] = true
	wd.ProbeOnce()
	st3, _ := store.TVState("tv-301-main")
	if st3.DeviceStatus != "idle" {
		t.Fatalf("expected tv-301-main to recover to idle, got %q", st3.DeviceStatus)
	}
}

func TestSelfUpdater_RejectsInvalidChecksumAndRollsBack(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "gateway")
	oldBinary := []byte("gateway-v1-binary")
	if err := os.WriteFile(binPath, oldBinary, 0o755); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	newBinary := []byte("gateway-v2-binary")
	sum := sha256.Sum256(newBinary)
	validHash := hex.EncodeToString(sum[:])

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(newBinary)
	}))
	defer srv.Close()

	// 1. Invalid checksum must be rejected before overwriting binary
	updater := NewSelfUpdater(binPath, srv.Client(), func() error { return nil })
	if err := updater.Apply(srv.URL+"/gateway", "0000badhash"); err == nil {
		t.Fatalf("expected error on invalid checksum")
	}
	got, _ := os.ReadFile(binPath)
	if !bytes.Equal(got, oldBinary) {
		t.Fatalf("binary was overwritten despite invalid checksum")
	}

	// 2. Valid checksum but failed healthcheck must trigger rollback
	failingUpdater := NewSelfUpdater(binPath, srv.Client(), func() error {
		return errors.New("healthcheck failed")
	})
	if err := failingUpdater.Apply(srv.URL+"/gateway", validHash); err == nil {
		t.Fatalf("expected error when healthcheck fails")
	}
	gotAfterRollback, _ := os.ReadFile(binPath)
	if !bytes.Equal(gotAfterRollback, oldBinary) {
		t.Fatalf("expected rollback to oldBinary, got %q", gotAfterRollback)
	}
}
