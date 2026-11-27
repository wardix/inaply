package assets

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestAssetCache_DownloadsAndVerifiesSHA256(t *testing.T) {
	validData := []byte("valid-video-payload-v1")
	validHash := sha256Hex(validData)
	corruptData := []byte("corrupted-payload")

	serveValid := true
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveValid {
			_, _ = w.Write(validData)
			return
		}
		_, _ = w.Write(corruptData)
	}))
	defer upstream.Close()

	cache := NewCache(t.TempDir(), upstream.Client())
	if err := cache.Ensure("promo.mp4", upstream.URL+"/promo.mp4", validHash); err != nil {
		t.Fatalf("Ensure valid asset failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(cache.Dir(), "promo.mp4"))
	if err != nil || !bytes.Equal(got, validData) {
		t.Fatalf("cached file mismatch: got=%q err=%v", got, err)
	}

	// Now upstream serves corrupted bytes for the same filename & expected validHash
	serveValid = false
	if err := cache.Ensure("promo.mp4", upstream.URL+"/promo.mp4", validHash); err == nil {
		t.Fatalf("expected error on SHA-256 mismatch")
	}

	// Existing valid cached file must NOT be overwritten
	gotAfter, err := os.ReadFile(filepath.Join(cache.Dir(), "promo.mp4"))
	if err != nil || !bytes.Equal(gotAfter, validData) {
		t.Fatalf("valid cached file was corrupted/overwritten: got=%q err=%v", gotAfter, err)
	}
}

func TestAssetHandler_SupportsHTTPByteRange(t *testing.T) {
	payload := bytes.Repeat([]byte("0123456789"), 20) // 200 bytes
	cache := NewCache(t.TempDir(), nil)
	if err := os.WriteFile(filepath.Join(cache.Dir(), "ad.mp4"), payload, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	handler := cache.Handler()
	req := httptest.NewRequest(http.MethodGet, "/assets/ad.mp4", nil)
	req.Header.Set("Range", "bytes=0-99")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("expected 206 Partial Content, got %d", rec.Code)
	}
	if rec.Body.Len() != 100 || !bytes.Equal(rec.Body.Bytes(), payload[:100]) {
		t.Fatalf("unexpected range body len=%d", rec.Body.Len())
	}
}

func TestAssetGC_RemovesOrphanedFiles(t *testing.T) {
	cache := NewCache(t.TempDir(), nil)
	for _, name := range []string{"active-ad.mp4", "ota-v2.apk", "orphan-old.mp4"} {
		if err := os.WriteFile(filepath.Join(cache.Dir(), name), []byte("data"), 0o644); err != nil {
			t.Fatalf("WriteFile %s failed: %v", name, err)
		}
	}

	removed, err := cache.GC([]string{"active-ad.mp4", "ota-v2.apk"})
	if err != nil {
		t.Fatalf("GC failed: %v", err)
	}
	if removed != 1 {
		t.Fatalf("expected 1 orphaned file removed, got %d", removed)
	}
	if _, err := os.Stat(filepath.Join(cache.Dir(), "orphan-old.mp4")); !os.IsNotExist(err) {
		t.Fatalf("expected orphan-old.mp4 to be deleted")
	}
	if _, err := os.Stat(filepath.Join(cache.Dir(), "active-ad.mp4")); err != nil {
		t.Fatalf("expected active-ad.mp4 to remain: %v", err)
	}
}
