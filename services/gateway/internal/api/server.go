package api

import (
	"encoding/json"
	"errors"
	"html/template"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/inaply/gateway/internal/assets"
	"github.com/inaply/gateway/internal/firewall"
	"github.com/inaply/gateway/internal/models"
	"github.com/inaply/gateway/internal/portal"
	"github.com/inaply/gateway/internal/session"
)

type Server struct {
	store          *session.Store
	fw             firewall.Driver
	mux            *http.ServeMux
	portalHandler  http.Handler
	portalSecret   string
	assetCache     *assets.Cache
	trustedProxies []*net.IPNet
}

func NewServer(store *session.Store, fw firewall.Driver) *Server {
	s := &Server{
		store: store,
		fw:    fw,
		mux:   http.NewServeMux(),
	}
	s.SetPortalWebhookSecret("")
	s.routes()
	return s
}

// SetAssetCache configures the local LAN asset cache served at /assets/.
func (s *Server) SetAssetCache(cache *assets.Cache) {
	s.assetCache = cache
}

// SetTrustedProxies configures CIDRs or single IPs whose X-Forwarded-For header is trusted.
func (s *Server) SetTrustedProxies(cidrsOrIPs []string) error {
	nets := make([]*net.IPNet, 0, len(cidrsOrIPs))
	for _, raw := range cidrsOrIPs {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, "/") {
			_, ipNet, err := net.ParseCIDR(trimmed)
			if err != nil {
				return err
			}
			nets = append(nets, ipNet)
			continue
		}
		ip := net.ParseIP(trimmed)
		if ip == nil {
			return errors.New("invalid trusted proxy IP: " + trimmed)
		}
		bits := 32
		if ip.To4() == nil {
			bits = 128
		}
		nets = append(nets, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
	}
	s.trustedProxies = nets
	return nil
}

// SetPortalWebhookSecret configures the shared secret for vendor portal webhooks.
func (s *Server) SetPortalWebhookSecret(secret string) {
	s.portalSecret = secret
	s.portalHandler = portal.NewWebhookHandler(
		s.store,
		portal.NewMikroTikAdapter(secret),
		portal.NewUniFiAdapter(secret),
		portal.NewRuijieAdapter(secret),
		portal.NewOmadaAdapter(secret),
	)
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("/healthz", s.handleHealth)
	s.mux.HandleFunc("/api/v1/pair/portal", s.handlePairPortal)
	s.mux.HandleFunc("/api/v1/guest/disconnect", s.handleGuestDisconnect)
	s.mux.HandleFunc("/api/v1/portal/webhook/", func(w http.ResponseWriter, r *http.Request) {
		s.portalHandler.ServeHTTP(w, r)
	})
	s.mux.HandleFunc("/api/v1/radius/accounting", s.handleRadiusAccounting)
	s.mux.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		if s.assetCache == nil {
			http.NotFound(w, r)
			return
		}
		s.assetCache.Handler().ServeHTTP(w, r)
	})
	s.mux.HandleFunc("/api/v1/pair/pin", s.handlePairPIN)
	s.mux.HandleFunc("/api/v1/tv/state", s.handleTVState)
	s.mux.HandleFunc("/api/v1/tv/heartbeat", s.handleTVHeartbeat)
	s.mux.HandleFunc("/api/v1/pms/checkin", s.handlePMSCheckIn)
	s.mux.HandleFunc("/api/v1/pms/checkout", s.handlePMSCheckOut)
	s.mux.HandleFunc("/api/v1/messages", s.handleMessages)
	s.mux.HandleFunc("/api/v1/ads", s.handleAds)
	s.mux.HandleFunc("/api/v1/gateway/status", s.handleGatewayStatus)

	// Lightweight interactive demo UIs for Captive Portal, QR/PIN Fallback, and Front Office
	s.mux.HandleFunc("/portal", s.handlePortalPage)
	s.mux.HandleFunc("/pair", s.handlePairPage)
	s.mux.HandleFunc("/fo", s.handleFrontOfficePage)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handlePairPortal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		RoomNumber string `json:"room_number"`
		GuestName  string `json:"guest_name"`
		GuestIP    string `json:"guest_ip"`
		GuestMAC   string `json:"guest_mac"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	guestIP := s.resolveClientIP(req.GuestIP, r)
	res, err := s.store.PairViaPortal(req.RoomNumber, req.GuestName, guestIP, req.GuestMAC)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handlePairPIN(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		PIN      string `json:"pin"`
		GuestIP  string `json:"guest_ip"`
		GuestMAC string `json:"guest_mac"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	guestIP := s.resolveClientIP(req.GuestIP, r)
	res, err := s.store.PairViaPIN(req.PIN, guestIP, req.GuestMAC)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleTVState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id query parameter is required")
		return
	}
	state, err := s.store.TVState(deviceID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) handleTVHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		DeviceID   string `json:"device_id"`
		Status     string `json:"status"`
		AppVersion string `json:"app_version"`
		HDMICEC    bool   `json:"hdmi_cec_attached"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if err := s.store.Heartbeat(req.DeviceID, req.Status, req.AppVersion, req.HDMICEC); err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handlePMSCheckIn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		RoomNumber string              `json:"room_number"`
		GuestName  string              `json:"guest_name"`
		Segment    models.GuestSegment `json:"segment"`
		CheckoutAt string              `json:"checkout_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	var outTime time.Time
	if req.CheckoutAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.CheckoutAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "checkout_at must be RFC3339")
			return
		}
		outTime = parsed
	}
	room, err := s.store.CheckIn(req.RoomNumber, req.GuestName, req.Segment, outTime)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, room)
}

func (s *Server) handlePMSCheckOut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		RoomNumber string `json:"room_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	cleared, err := s.store.CheckOut(req.RoomNumber)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"room_number":      req.RoomNumber,
		"sessions_cleared": cleared,
	})
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		RoomNumber    string                 `json:"room_number"`
		Title         string                 `json:"title"`
		Body          string                 `json:"body"`
		Priority      models.MessagePriority `json:"priority"`
		ShowAsOverlay bool                   `json:"show_as_overlay"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Body) == "" {
		writeError(w, http.StatusBadRequest, "title and body are required")
		return
	}
	msg := s.store.AddMessage(models.GuestMessage{
		RoomNumber:    strings.TrimSpace(req.RoomNumber),
		Title:         strings.TrimSpace(req.Title),
		Body:          strings.TrimSpace(req.Body),
		Priority:      req.Priority,
		ShowAsOverlay: req.ShowAsOverlay,
	})
	writeJSON(w, http.StatusOK, msg)
}

func (s *Server) handleAds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req models.AdCampaign
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.MediaURL) == "" {
		writeError(w, http.StatusBadRequest, "title and media_url are required")
		return
	}
	ad := s.store.AddAd(req)
	writeJSON(w, http.StatusOK, ad)
}

func (s *Server) handleGatewayStatus(w http.ResponseWriter, r *http.Request) {
	rooms, sessions, receivers := s.store.ListRoomsAndSessions()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"rooms":          rooms,
		"sessions":       sessions,
		"receivers":      receivers,
		"firewall_rules": s.fw.ActiveRules(),
	})
}

func (s *Server) handlePortalPage(w http.ResponseWriter, r *http.Request) {
	renderPage(w, portalHTML, map[string]string{
		"GuestIP": s.resolveClientIP("", r),
	})
}

func (s *Server) handlePairPage(w http.ResponseWriter, r *http.Request) {
	renderPage(w, pairHTML, map[string]string{
		"GuestIP": s.resolveClientIP("", r),
		"PIN":     r.URL.Query().Get("pin"),
	})
}

func (s *Server) handleFrontOfficePage(w http.ResponseWriter, r *http.Request) {
	rooms, sessions, _ := s.store.ListRoomsAndSessions()
	renderPage(w, foHTML, map[string]interface{}{
		"Rooms":    rooms,
		"Sessions": sessions,
		"Rules":    s.fw.ActiveRules(),
	})
}

func (s *Server) isTrustedProxy(remoteAddr string) bool {
	if len(s.trustedProxies) == 0 {
		return false
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil || host == "" {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, ipNet := range s.trustedProxies {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

func (s *Server) resolveClientIP(explicit string, r *http.Request) string {
	if ip := strings.TrimSpace(explicit); ip != "" {
		return ip
	}
	if s.isTrustedProxy(r.RemoteAddr) {
		if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
			parts := strings.Split(xff, ",")
			if p := strings.TrimSpace(parts[0]); net.ParseIP(p) != nil {
				return p
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		if host == "::1" {
			return "127.0.0.1"
		}
		return host
	}
	if r.RemoteAddr == "::1" {
		return "127.0.0.1"
	}
	return r.RemoteAddr
}

func (s *Server) handleRadiusAccounting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.portalSecret != "" {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) || strings.TrimSpace(strings.TrimPrefix(auth, prefix)) != s.portalSecret {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
	}
	var ev models.RadiusAccountingEvent
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := s.store.HandleRadiusAccounting(ev); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, session.ErrRoomNotFound), errors.Is(err, session.ErrDeviceNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, session.ErrGuestMismatch), errors.Is(err, session.ErrInvalidPIN):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, session.ErrInvalidGuestIP), errors.Is(err, session.ErrRoomNotConfigured):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func renderPage(w http.ResponseWriter, tplStr string, data interface{}) {
	tpl, err := template.New("page").Parse(tplStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tpl.Execute(w, data)
}

const portalHTML = `<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Hotel Wi-Fi Portal & Auto TV Pairing</title>
<style>body{font-family:system-ui,sans-serif;max-width:440px;margin:40px auto;padding:0 16px;background:#0f172a;color:#f8fafc}
.card{background:#1e293b;padding:24px;border-radius:12px}input,button{width:100%;padding:12px;margin:8px 0;border-radius:8px;border:1px solid #334155;box-sizing:border-box}
button{background:#0284c7;color:#fff;font-weight:600;border:none;cursor:pointer}pre{background:#020617;padding:12px;border-radius:8px;overflow:auto}</style></head>
<body><div class="card"><h2>Hotel Wi-Fi Login</h2>
<p>IP Anda: <b>{{.GuestIP}}</b>. Masuk untuk terhubung ke internet & otomatis pairing ke TV kamar Anda.</p>
<form id="f"><input name="room_number" placeholder="Nomor Kamar (mis. 301)" required>
<input name="guest_name" placeholder="Nama Tamu (mis. Budi)" required>
<button type="submit">Connect & Auto-Pair TV</button></form><pre id="out"></pre></div>
<script>
document.getElementById('f').onsubmit = async (e) => {
  e.preventDefault();
  const fd = new FormData(e.target);
  const res = await fetch('/api/v1/pair/portal', {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({room_number: fd.get('room_number'), guest_name: fd.get('guest_name')})
  });
  document.getElementById('out').textContent = JSON.stringify(await res.json(), null, 2);
};
</script></body></html>`

const pairHTML = `<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Fallback QR / PIN TV Pairing</title>
<style>body{font-family:system-ui,sans-serif;max-width:440px;margin:40px auto;padding:0 16px;background:#0f172a;color:#f8fafc}
.card{background:#1e293b;padding:24px;border-radius:12px}input,button{width:100%;padding:12px;margin:8px 0;border-radius:8px;border:1px solid #334155;box-sizing:border-box}
button{background:#059669;color:#fff;font-weight:600;border:none;cursor:pointer}pre{background:#020617;padding:12px;border-radius:8px;overflow:auto}</style></head>
<body><div class="card"><h2>Pair ke TV Kamar (Fallback PIN)</h2>
<p>IP Anda: <b>{{.GuestIP}}</b>. Masukkan 6 digit PIN yang tampil di layar TV.</p>
<form id="f"><input name="pin" value="{{.PIN}}" placeholder="6-digit PIN" required>
<button type="submit">Pair Device</button></form><pre id="out"></pre></div>
<script>
document.getElementById('f').onsubmit = async (e) => {
  e.preventDefault();
  const fd = new FormData(e.target);
  const res = await fetch('/api/v1/pair/pin', {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({pin: fd.get('pin')})
  });
  document.getElementById('out').textContent = JSON.stringify(await res.json(), null, 2);
};
</script></body></html>`

const foHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>Inaply Front Office Console</title>
<style>body{font-family:system-ui,sans-serif;max-width:900px;margin:32px auto;padding:0 16px;background:#f8fafc;color:#0f172a}
table{width:100%;border-collapse:collapse;background:#fff;margin-bottom:24px}th,td{padding:10px;border:1px solid #cbd5e1;text-align:left}
button{padding:6px 12px;cursor:pointer}</style></head>
<body><h1>Front Office & Gateway Status</h1>
<h3>Kamar</h3><table><tr><th>Kamar</th><th>Status</th><th>Tamu</th><th>Segmen</th><th>PIN TV</th><th>Aksi</th></tr>
{{range .Rooms}}<tr><td>{{.RoomNumber}}</td><td>{{if .Occupied}}Occupied{{else}}Vacant{{end}}</td><td>{{.GuestName}}</td><td>{{.Segment}}</td><td><code>{{.CurrentPIN}}</code></td>
<td><button onclick="checkout('{{.RoomNumber}}')">Check-out & Clear</button></td></tr>{{end}}</table>
<h3>Sesi Tamu Aktif</h3><table><tr><th>Guest IP</th><th>Kamar</th><th>Mode</th><th>Tamu</th></tr>
{{range .Sessions}}<tr><td>{{.GuestIP}}</td><td>{{.RoomNumber}}</td><td>{{.PairingMode}}</td><td>{{.GuestName}}</td></tr>{{end}}</table>
<h3>Aturan Firewall L3 Aktif</h3><table><tr><th>Guest IP</th><th>Allowed TV IP</th></tr>
{{range .Rules}}<tr><td>{{.GuestIP}}</td><td>{{.ReceiverIP}}</td></tr>{{end}}</table>
<script>
async function checkout(room){
  await fetch('/api/v1/pms/checkout',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({room_number:room})});
  location.reload();
}
</script></body></html>`

func (s *Server) handleGuestDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		GuestIP string `json:"guest_ip"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	callerIP := s.resolveClientIP("", r)
	if req.GuestIP != "" && req.GuestIP != callerIP {
		writeError(w, http.StatusForbidden, "cannot disconnect another device")
		return
	}
	if err := s.store.DisconnectGuestIP(callerIP); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected", "guest_ip": callerIP})
}
