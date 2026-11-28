package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/inaply/gateway/internal/firewall"
	"github.com/inaply/gateway/internal/models"
	"github.com/inaply/gateway/internal/session"
)

func TestEndToEndHTTPWorkflow(t *testing.T) {
	fw := firewall.NewMemoryDriver()
	store := session.NewStore(fw, nil)
	store.SeedDemoProperty()

	srv := NewServer(store, fw)
	srv.SetAdminSecret("test-admin-secret")
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
	msgReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/messages", bytes.NewReader(msgBody))
	msgReq.Header.Set("Content-Type", "application/json")
	msgReq.Header.Set("Authorization", "Bearer test-admin-secret")
	msgRes, err := http.DefaultClient.Do(msgReq)
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
	coReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/pms/checkout", bytes.NewReader(coBody))
	coReq.Header.Set("Content-Type", "application/json")
	coReq.Header.Set("Authorization", "Bearer test-admin-secret")
	coRes, err := http.DefaultClient.Do(coReq)
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

	// 1. Cross-IP disconnect attempt must be rejected with 403 Forbidden
	crossReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/guest/disconnect",
		bytes.NewBufferString(`{"guest_ip":"10.10.30.20"}`),
	)
	crossReq.RemoteAddr = "10.10.30.10:51234"
	crossReq.Header.Set("Content-Type", "application/json")
	crossRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(crossRec, crossReq)
	if crossRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for cross-IP disconnect, got %d", crossRec.Code)
	}
	if room, _ := store.ReceiversForGuestIP("10.10.30.20"); room == nil || room.RoomNumber != "302" {
		t.Fatalf("expected 10.10.30.20 in room 302 to remain connected after 403")
	}

	// 2. Self-disconnect using RemoteAddr (with TCP port) disconnects only caller IP
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/guest/disconnect",
		bytes.NewBufferString(`{}`),
	)
	req.RemoteAddr = "10.10.30.10:51234"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
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

func TestResolveClientIP_IgnoresSpoofedXForwardedForFromUntrustedPeer(t *testing.T) {
	fw := firewall.NewMemoryDriver()
	store := session.NewStore(fw, nil)
	srv := NewServer(store, fw)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/guest/disconnect", nil)
	req.RemoteAddr = "10.10.30.10:1234"
	req.Header.Set("X-Forwarded-For", "10.10.30.20")

	if got := srv.resolveClientIP("", req); got != "10.10.30.10" {
		t.Fatalf("expected untrusted peer X-Forwarded-For to be ignored and return 10.10.30.10, got %q", got)
	}

	if err := srv.SetTrustedProxies([]string{"10.10.30.0/24"}); err != nil {
		t.Fatalf("SetTrustedProxies failed: %v", err)
	}
	if got := srv.resolveClientIP("", req); got != "10.10.30.20" {
		t.Fatalf("expected trusted proxy X-Forwarded-For to be accepted and return 10.10.30.20, got %q", got)
	}
}

func TestRadiusAccountingWebhook_AuthAndSync(t *testing.T) {
	fw := firewall.NewMemoryDriver()
	store := session.NewStore(fw, nil)
	srv := NewServer(store, fw)
	srv.SetPortalWebhookSecret("radius-secret")

	// Seed room 301 and guest session
	store.SeedDemoProperty()
	store.CheckIn("301", "Alice", models.SegmentVIP, time.Time{})
	if _, err := store.PairViaPortal("301", "Alice", "10.10.30.10", "aa:bb:cc:dd:ee:01"); err != nil {
		t.Fatalf("PairViaPortal failed: %v", err)
	}
	if err := store.HandleRadiusAccounting(models.RadiusAccountingEvent{
		StatusType:      "Start",
		RadiusSessionID: "rad-301-1",
		GuestIP:         "10.10.30.10",
		GuestMAC:        "aa:bb:cc:dd:ee:01",
		RoomNumber:      "301",
	}); err != nil {
		t.Fatalf("initial HandleRadiusAccounting failed: %v", err)
	}

	body := `{"acct_status_type":"Interim-Update","acct_session_id":"rad-301-1","framed_ip_address":"10.10.30.99","calling_station_id":"aa:bb:cc:dd:ee:01","room_number":"301"}`

	// 1. Wrong token -> 401
	reqUnauth := httptest.NewRequest(http.MethodPost, "/api/v1/radius/accounting", bytes.NewBufferString(body))
	reqUnauth.Header.Set("Authorization", "Bearer wrong-secret")
	recUnauth := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recUnauth, reqUnauth)
	if recUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", recUnauth.Code)
	}

	// 2. Valid token -> 200 and roaming IP updated
	reqAuth := httptest.NewRequest(http.MethodPost, "/api/v1/radius/accounting", bytes.NewBufferString(body))
	reqAuth.Header.Set("Authorization", "Bearer radius-secret")
	reqAuth.Header.Set("Content-Type", "application/json")
	recAuth := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recAuth, reqAuth)
	if recAuth.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (%s)", recAuth.Code, recAuth.Body.String())
	}

	room, _ := store.ReceiversForGuestIP("10.10.30.99")
	if room == nil || room.RoomNumber != "301" {
		t.Fatalf("expected roaming IP 10.10.30.99 to be mapped to room 301, got %+v", room)
	}

	// 3. Empty secret -> fail-closed 401 Unauthorized
	srv.SetPortalWebhookSecret("")
	reqEmptySecret := httptest.NewRequest(http.MethodPost, "/api/v1/radius/accounting", bytes.NewBufferString(body))
	reqEmptySecret.Header.Set("Authorization", "Bearer ")
	recEmptySecret := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recEmptySecret, reqEmptySecret)
	if recEmptySecret.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized when portalSecret is empty, got %d", recEmptySecret.Code)
	}
}

func TestPairViaPortal_SignageDeviceReturns403(t *testing.T) {
	fw := firewall.NewMemoryDriver()
	store := session.NewStore(fw, nil)
	store.SeedDemoProperty()
	store.CheckIn("301", "Alice", models.SegmentVIP, time.Time{})
	store.UpsertReceiver(models.ReceiverDevice{
		DeviceID:    "tv-lobby-1",
		RoomNumber:  "301",
		Label:       "Lobby Signage",
		IP:          "10.10.20.99",
		Mode:        "signage",
		Orientation: "landscape",
		Status:      "online",
	})

	srv := NewServer(store, fw)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/pair/portal",
		bytes.NewBufferString(`{"room_number":"301","guest_name":"Alice"}`),
	)
	req.RemoteAddr = "10.10.30.10:1234"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for signage device pairing, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestAdminAuth_RejectsMissingOrWrongTokenOnPMSAndFOEndpoints(t *testing.T) {
	fw := firewall.NewMemoryDriver()
	store := session.NewStore(fw, nil)
	store.SeedDemoProperty()
	srv := NewServer(store, fw)

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/v1/pms/checkin", `{"room_number":"301","guest_name":"Alice"}`},
		{http.MethodPost, "/api/v1/pms/checkout", `{"room_number":"301"}`},
		{http.MethodPost, "/api/v1/messages", `{"title":"Hi","body":"Welcome","priority":"normal"}`},
		{http.MethodPost, "/api/v1/ads", `{"id":"ad-1","title":"Spa","media_url":"http://x/a.mp4","duration_sec":10,"active":true}`},
		{http.MethodGet, "/api/v1/gateway/status", ""},
		{http.MethodGet, "/fo", ""},
	}

	// 1. Empty adminSecret -> fail-closed 401
	for _, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString(ep.body))
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 when adminSecret is empty for %s %s, got %d", ep.method, ep.path, rec.Code)
		}
	}

	// 2. Configured adminSecret, missing or wrong token -> 401
	srv.SetAdminSecret("super-admin-secret")
	for _, ep := range endpoints {
		reqMissing := httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString(ep.body))
		recMissing := httptest.NewRecorder()
		srv.Handler().ServeHTTP(recMissing, reqMissing)
		if recMissing.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for missing token on %s %s, got %d", ep.method, ep.path, recMissing.Code)
		}

		reqWrong := httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString(ep.body))
		reqWrong.Header.Set("Authorization", "Bearer wrong-secret")
		recWrong := httptest.NewRecorder()
		srv.Handler().ServeHTTP(recWrong, reqWrong)
		if recWrong.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for wrong token on %s %s, got %d", ep.method, ep.path, recWrong.Code)
		}
	}
}

func TestAdminAuth_AllowsValidTokenAndKeepsGuestTVRoutesWorking(t *testing.T) {
	fw := firewall.NewMemoryDriver()
	store := session.NewStore(fw, nil)
	store.SeedDemoProperty()
	srv := NewServer(store, fw)
	srv.SetAdminSecret("super-admin-secret")

	// Valid Bearer token -> 200 OK on admin endpoints
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/v1/gateway/status", nil)
	reqStatus.Header.Set("Authorization", "Bearer super-admin-secret")
	recStatus := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with valid admin token, got %d", recStatus.Code)
	}

	// Guest & TV routes remain accessible without admin token
	for _, path := range []string{"/healthz", "/api/v1/tv/state?device_id=tv-301-main"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK without admin token for %s, got %d", path, rec.Code)
		}
	}
}
