package portal

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/inaply/gateway/internal/session"
)

var (
	ErrUnauthorized   = errors.New("invalid portal webhook token")
	ErrInvalidPayload = errors.New("invalid portal webhook payload")
)

// PortalEvent is the normalized captive portal login event passed to PairViaPortal.
type PortalEvent struct {
	RoomNumber string
	GuestName  string
	GuestIP    string
	GuestMAC   string
}

// PortalAdapter normalizes vendor-specific Wi-Fi controller / captive portal webhooks.
type PortalAdapter interface {
	Vendor() string
	ParseWebhook(headers http.Header, body []byte) (PortalEvent, error)
}

type tokenValidator struct {
	secret string
}

func (v tokenValidator) validate(headers http.Header) error {
	if v.secret == "" {
		return nil
	}
	token := headers.Get("X-Inaply-Token")
	if token == "" {
		auth := headers.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(v.secret)) != 1 {
		return ErrUnauthorized
	}
	return nil
}

// MikroTikAdapter handles MikroTik Hotspot login webhooks.
type MikroTikAdapter struct {
	tokenValidator
}

func NewMikroTikAdapter(secret string) *MikroTikAdapter {
	return &MikroTikAdapter{tokenValidator: tokenValidator{secret: secret}}
}

func (a *MikroTikAdapter) Vendor() string { return "mikrotik" }

func (a *MikroTikAdapter) ParseWebhook(headers http.Header, body []byte) (PortalEvent, error) {
	if err := a.validate(headers); err != nil {
		return PortalEvent{}, err
	}
	var p struct {
		Room      string `json:"room"`
		GuestName string `json:"guest_name"`
		IP        string `json:"ip"`
		MAC       string `json:"mac"`
	}
	if err := json.Unmarshal(body, &p); err != nil || p.Room == "" || p.IP == "" {
		return PortalEvent{}, ErrInvalidPayload
	}
	return PortalEvent{
		RoomNumber: p.Room,
		GuestName:  p.GuestName,
		GuestIP:    p.IP,
		GuestMAC:   p.MAC,
	}, nil
}

// UniFiAdapter handles Ubiquiti UniFi Network external portal webhooks.
type UniFiAdapter struct {
	tokenValidator
}

func NewUniFiAdapter(secret string) *UniFiAdapter {
	return &UniFiAdapter{tokenValidator: tokenValidator{secret: secret}}
}

func (a *UniFiAdapter) Vendor() string { return "unifi" }

func (a *UniFiAdapter) ParseWebhook(headers http.Header, body []byte) (PortalEvent, error) {
	if err := a.validate(headers); err != nil {
		return PortalEvent{}, err
	}
	var p struct {
		RoomNumber string `json:"room_number"`
		GuestName  string `json:"guest_name"`
		ClientIP   string `json:"client_ip"`
		ClientMAC  string `json:"client_mac"`
	}
	if err := json.Unmarshal(body, &p); err != nil || p.RoomNumber == "" || p.ClientIP == "" {
		return PortalEvent{}, ErrInvalidPayload
	}
	return PortalEvent{
		RoomNumber: p.RoomNumber,
		GuestName:  p.GuestName,
		GuestIP:    p.ClientIP,
		GuestMAC:   p.ClientMAC,
	}, nil
}

// RuijieAdapter handles Ruijie Cloud/Gateway captive portal webhooks.
type RuijieAdapter struct {
	tokenValidator
}

func NewRuijieAdapter(secret string) *RuijieAdapter {
	return &RuijieAdapter{tokenValidator: tokenValidator{secret: secret}}
}

func (a *RuijieAdapter) Vendor() string { return "ruijie" }

func (a *RuijieAdapter) ParseWebhook(headers http.Header, body []byte) (PortalEvent, error) {
	if err := a.validate(headers); err != nil {
		return PortalEvent{}, err
	}
	var p struct {
		RoomNo    string `json:"roomNo"`
		GuestName string `json:"guestName"`
		UserIP    string `json:"userIp"`
		UserMAC   string `json:"userMac"`
	}
	if err := json.Unmarshal(body, &p); err != nil || p.RoomNo == "" || p.UserIP == "" {
		return PortalEvent{}, ErrInvalidPayload
	}
	return PortalEvent{
		RoomNumber: p.RoomNo,
		GuestName:  p.GuestName,
		GuestIP:    p.UserIP,
		GuestMAC:   p.UserMAC,
	}, nil
}

// OmadaAdapter handles TP-Link Omada Controller external portal webhooks.
type OmadaAdapter struct {
	tokenValidator
}

func NewOmadaAdapter(secret string) *OmadaAdapter {
	return &OmadaAdapter{tokenValidator: tokenValidator{secret: secret}}
}

func (a *OmadaAdapter) Vendor() string { return "omada" }

func (a *OmadaAdapter) ParseWebhook(headers http.Header, body []byte) (PortalEvent, error) {
	if err := a.validate(headers); err != nil {
		return PortalEvent{}, err
	}
	var p struct {
		Room      string `json:"room"`
		Name      string `json:"name"`
		ClientIP  string `json:"clientIp"`
		ClientMAC string `json:"clientMac"`
	}
	if err := json.Unmarshal(body, &p); err != nil || p.Room == "" || p.ClientIP == "" {
		return PortalEvent{}, ErrInvalidPayload
	}
	return PortalEvent{
		RoomNumber: p.Room,
		GuestName:  p.Name,
		GuestIP:    p.ClientIP,
		GuestMAC:   p.ClientMAC,
	}, nil
}

// NewWebhookHandler routes `/api/v1/portal/webhook/<vendor>` to the matching PortalAdapter
// and delegates pairing uniformly to store.PairViaPortal.
func NewWebhookHandler(store *session.Store, adapters ...PortalAdapter) http.Handler {
	byVendor := make(map[string]PortalAdapter, len(adapters))
	for _, a := range adapters {
		byVendor[strings.ToLower(a.Vendor())] = a
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		vendor := strings.ToLower(strings.TrimPrefix(r.URL.Path, "/api/v1/portal/webhook/"))
		adapter, ok := byVendor[vendor]
		if !ok {
			http.Error(w, "unknown portal vendor", http.StatusNotFound)
			return
		}

		var buf [64 * 1024]byte
		n, _ := r.Body.Read(buf[:])
		ev, err := adapter.ParseWebhook(r.Header, buf[:n])
		if err != nil {
			if errors.Is(err, ErrUnauthorized) {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		res, err := store.PairViaPortal(ev.RoomNumber, ev.GuestName, ev.GuestIP, ev.GuestMAC)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	})
}
