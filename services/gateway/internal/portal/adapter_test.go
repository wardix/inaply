package portal

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/inaply/gateway/internal/firewall"
	"github.com/inaply/gateway/internal/session"
)

func setupTestPortalHandler(secret string) (http.Handler, *session.Store, *firewall.MemoryDriver) {
	fw := firewall.NewMemoryDriver()
	store := session.NewStore(fw, nil)
	store.SeedDemoProperty()

	h := NewWebhookHandler(store,
		NewMikroTikAdapter(secret),
		NewUniFiAdapter(secret),
		NewRuijieAdapter(secret),
		NewOmadaAdapter(secret),
	)
	return h, store, fw
}

func TestPortalAdapter_MikroTikWebhook_BindsGuestIPToRoom(t *testing.T) {
	h, store, fw := setupTestPortalHandler("super-secret")

	body := []byte(`{"room":"301","guest_name":"Budi","ip":"10.10.30.10","mac":"aa:bb:cc:00:00:01"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portal/webhook/mikrotik", bytes.NewReader(body))
	req.Header.Set("X-Inaply-Token", "super-secret")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	room, rcvs := store.ReceiversForGuestIP("10.10.30.10")
	if room == nil || room.RoomNumber != "301" || len(rcvs) == 0 {
		t.Fatalf("expected guest IP 10.10.30.10 bound to room 301")
	}
	if len(fw.ActiveRules()) == 0 {
		t.Fatalf("expected firewall rule to be created")
	}
}

func TestPortalAdapter_UniFiAndOmada_NormalizesPayloads(t *testing.T) {
	cases := []struct {
		vendor  string
		payload string
		wantIP  string
		room    string
	}{
		{
			vendor:  "unifi",
			payload: `{"room_number":"301","guest_name":"Budi","client_ip":"10.10.30.21","client_mac":"aa:bb:cc:00:00:11"}`,
			wantIP:  "10.10.30.21",
			room:    "301",
		},
		{
			vendor:  "ruijie",
			payload: `{"roomNo":"302","guestName":"Siti","userIp":"10.10.30.22","userMac":"aa:bb:cc:00:00:12"}`,
			wantIP:  "10.10.30.22",
			room:    "302",
		},
		{
			vendor:  "omada",
			payload: `{"room":"301","name":"Budi","clientIp":"10.10.30.23","clientMac":"aa:bb:cc:00:00:13"}`,
			wantIP:  "10.10.30.23",
			room:    "301",
		},
	}

	for _, tc := range cases {
		t.Run(tc.vendor, func(t *testing.T) {
			h, store, _ := setupTestPortalHandler("secret-123")
			req := httptest.NewRequest(http.MethodPost, "/api/v1/portal/webhook/"+tc.vendor, bytes.NewBufferString(tc.payload))
			req.Header.Set("X-Inaply-Token", "secret-123")
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("%s: expected 200 OK, got %d: %s", tc.vendor, rec.Code, rec.Body.String())
			}

			room, _ := store.ReceiversForGuestIP(tc.wantIP)
			if room == nil || room.RoomNumber != tc.room {
				t.Fatalf("%s: expected IP %s bound to room %s", tc.vendor, tc.wantIP, tc.room)
			}
		})
	}
}

func TestPortalAdapter_RejectsInvalidSignatureOrUnknownRoom(t *testing.T) {
	t.Run("invalid secret returns 401", func(t *testing.T) {
		h, store, fw := setupTestPortalHandler("correct-secret")
		body := []byte(`{"room":"301","guest_name":"Budi","ip":"10.10.30.99"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/portal/webhook/mikrotik", bytes.NewReader(body))
		req.Header.Set("X-Inaply-Token", "wrong-secret")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
		}
		if room, _ := store.ReceiversForGuestIP("10.10.30.99"); room != nil {
			t.Fatalf("expected no session created on 401")
		}
		if len(fw.ActiveRules()) != 0 {
			t.Fatalf("expected no firewall rules on 401")
		}
	})

	t.Run("unknown room creates no session", func(t *testing.T) {
		h, store, fw := setupTestPortalHandler("correct-secret")
		body := []byte(`{"room":"999","guest_name":"Ghost","ip":"10.10.30.98"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/portal/webhook/mikrotik", bytes.NewReader(body))
		req.Header.Set("X-Inaply-Token", "correct-secret")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)
		if rec.Code == http.StatusOK {
			t.Fatalf("expected non-200 for unknown room 999")
		}
		if room, _ := store.ReceiversForGuestIP("10.10.30.98"); room != nil {
			t.Fatalf("expected no session created for unknown room")
		}
		if len(fw.ActiveRules()) != 0 {
			t.Fatalf("expected no firewall rules for unknown room")
		}
	})

	t.Run("empty secret fails closed with 401", func(t *testing.T) {
		h, _, _ := setupTestPortalHandler("")
		body := []byte(`{"room":"301","guest_name":"Budi","ip":"10.10.30.97"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/portal/webhook/mikrotik", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized when secret is empty, got %d", rec.Code)
		}
	})

	t.Run("oversized body returns 413", func(t *testing.T) {
		h, _, _ := setupTestPortalHandler("correct-secret")
		oversized := bytes.Repeat([]byte("a"), maxWebhookBodyBytes+10)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/portal/webhook/mikrotik", bytes.NewReader(oversized))
		req.Header.Set("X-Inaply-Token", "correct-secret")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected 413 Request Entity Too Large, got %d", rec.Code)
		}
	})
}
