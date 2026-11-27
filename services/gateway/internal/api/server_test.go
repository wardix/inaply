package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/inaply/gateway/internal/firewall"
	"github.com/inaply/gateway/internal/models"
	"github.com/inaply/gateway/internal/session"
)

func TestEndToEndHTTPWorkflow(t *testing.T) {
	fw := firewall.NewMemoryDriver()
	store := session.NewStore(fw, nil)
	store.SeedDemoProperty()

	srv := NewServer(store, fw)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// 1. TV fetches initial state & PIN
	res, err := http.Get(ts.URL + "/api/v1/tv/state?device_id=tv-301-main")
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("GET /tv/state failed: %v status=%v", err, res)
	}
	var tvState models.TVBackdropState
	if err := json.NewDecoder(res.Body).Decode(&tvState); err != nil {
		t.Fatalf("decode tv state failed: %v", err)
	}
	_ = res.Body.Close()
	if tvState.PairingPIN == "" || tvState.RoomNumber != "301" {
		t.Fatalf("unexpected tv state: %+v", tvState)
	}

	// 2. Guest pairs via fallback PIN
	pinBody, _ := json.Marshal(map[string]string{
		"pin":      tvState.PairingPIN,
		"guest_ip": "10.10.30.45",
	})
	pairRes, err := http.Post(ts.URL+"/api/v1/pair/pin", "application/json", bytes.NewReader(pinBody))
	if err != nil || pairRes.StatusCode != http.StatusOK {
		t.Fatalf("POST /pair/pin failed: %v", err)
	}
	_ = pairRes.Body.Close()

	// 3. Push overlay message to Room 301
	msgBody, _ := json.Marshal(map[string]interface{}{
		"room_number":     "301",
		"title":           "Late Checkout Confirmed",
		"body":            "Extended until 14:00.",
		"priority":        "normal",
		"show_as_overlay": true,
	})
	msgRes, err := http.Post(ts.URL+"/api/v1/messages", "application/json", bytes.NewReader(msgBody))
	if err != nil || msgRes.StatusCode != http.StatusOK {
		t.Fatalf("POST /messages failed: %v", err)
	}
	_ = msgRes.Body.Close()

	// 4. Verify TV sees active_paired_devices=1 and the new message
	res2, _ := http.Get(ts.URL + "/api/v1/tv/state?device_id=tv-301-main")
	var tvState2 models.TVBackdropState
	_ = json.NewDecoder(res2.Body).Decode(&tvState2)
	_ = res2.Body.Close()
	if tvState2.ActivePairedDevices != 1 || len(tvState2.Messages) < 2 {
		t.Fatalf("expected 1 paired device and >=2 messages, got %+v", tvState2)
	}

	// 5. Front Office triggers checkout -> clears session & increments ResetEpoch
	coBody, _ := json.Marshal(map[string]string{"room_number": "301"})
	coRes, err := http.Post(ts.URL+"/api/v1/pms/checkout", "application/json", bytes.NewReader(coBody))
	if err != nil || coRes.StatusCode != http.StatusOK {
		t.Fatalf("POST /pms/checkout failed: %v", err)
	}
	_ = coRes.Body.Close()

	if len(fw.ActiveRules()) != 0 {
		t.Fatalf("expected 0 firewall rules after checkout, got %+v", fw.ActiveRules())
	}

	res3, _ := http.Get(ts.URL + "/api/v1/tv/state?device_id=tv-301-main")
	var tvState3 models.TVBackdropState
	_ = json.NewDecoder(res3.Body).Decode(&tvState3)
	_ = res3.Body.Close()
	if tvState3.ResetEpoch <= tvState2.ResetEpoch || tvState3.ActivePairedDevices != 0 {
		t.Fatalf("expected ResetEpoch bump and 0 paired devices after checkout, got %+v", tvState3)
	}
}

func TestPortalWebhookRouteIntegration(t *testing.T) {
	fw := firewall.NewMemoryDriver()
	store := session.NewStore(fw, nil)
	store.SeedDemoProperty()

	srv := NewServer(store, fw)
	srv.SetPortalWebhookSecret("integration-secret")
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, err := http.NewRequest(
		http.MethodPost,
		ts.URL+"/api/v1/portal/webhook/mikrotik",
		bytes.NewBufferString(`{"room":"301","guest_name":"Budi","ip":"10.10.30.77","mac":"aa:bb:cc:00:00:77"}`),
	)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	req.Header.Set("X-Inaply-Token", "integration-secret")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/v1/portal/webhook/mikrotik, got %d", resp.StatusCode)
	}
	if room, _ := store.ReceiversForGuestIP("10.10.30.77"); room == nil || room.RoomNumber != "301" {
		t.Fatalf("expected 10.10.30.77 bound to room 301")
	}
}

func TestGuestSelfService_DisconnectsOnlyOwnDevice(t *testing.T) {
	fw := firewall.NewMemoryDriver()
	store := session.NewStore(fw, nil)
	store.SeedDemoProperty()

	// Bind two devices in room 301 and one in room 302
	if _, err := store.PairViaPortal("301", "Budi", "10.10.30.10", "aa:bb:cc:00:00:01"); err != nil {
		t.Fatalf("PairViaPortal 301-A failed: %v", err)
	}
	if _, err := store.PairViaPortal("301", "Budi", "10.10.30.11", "aa:bb:cc:00:00:02"); err != nil {
		t.Fatalf("PairViaPortal 301-B failed: %v", err)
	}
	if _, err := store.PairViaPortal("302", "Siti", "10.10.30.20", "aa:bb:cc:00:00:03"); err != nil {
		t.Fatalf("PairViaPortal 302 failed: %v", err)
	}

	srv := NewServer(store, fw)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, err := http.NewRequest(
		http.MethodPost,
		ts.URL+"/api/v1/guest/disconnect",
		bytes.NewBufferString(`{"guest_ip":"10.10.30.10"}`),
	)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	if room, _ := store.ReceiversForGuestIP("10.10.30.10"); room != nil {
		t.Fatalf("expected 10.10.30.10 to be disconnected")
	}
	if room, _ := store.ReceiversForGuestIP("10.10.30.11"); room == nil || room.RoomNumber != "301" {
		t.Fatalf("expected 10.10.30.11 in room 301 to remain connected")
	}
	if room, _ := store.ReceiversForGuestIP("10.10.30.20"); room == nil || room.RoomNumber != "302" {
		t.Fatalf("expected 10.10.30.20 in room 302 to remain connected")
	}
}
