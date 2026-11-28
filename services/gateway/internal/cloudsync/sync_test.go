package cloudsync

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCloudSyncAuth(t *testing.T) {
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Gateway-Token") != "valid-prop-a-token" || r.Header.Get("X-Property-ID") != "prop-a" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"seq":1,"ads":[],"messages":[],"force_checkout_rooms":[]}`))
	}))
	defer cloud.Close()

	// Invalid token -> 401 error
	badClient := NewClient(cloud.URL, "prop-a", "wrong-token", cloud.Client())
	if _, err := badClient.PullDelta(0); err == nil {
		t.Fatalf("expected 401 error for invalid token")
	}

	// Cross-property token -> 401 error
	crossClient := NewClient(cloud.URL, "prop-b", "valid-prop-a-token", cloud.Client())
	if _, err := crossClient.PullDelta(0); err == nil {
		t.Fatalf("expected 401 error for cross-property mismatch")
	}

	// Valid token & property -> 200 OK
	goodClient := NewClient(cloud.URL, "prop-a", "valid-prop-a-token", cloud.Client())
	delta, err := goodClient.PullDelta(0)
	if err != nil || delta.Seq != 1 {
		t.Fatalf("expected valid delta seq=1, got delta=%+v err=%v", delta, err)
	}
}
