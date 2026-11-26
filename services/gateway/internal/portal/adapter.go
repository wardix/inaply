package portal

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/inaply/gateway/internal/session"
)

const maxWebhookBodyBytes = 64 * 1024

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
	// Fail-closed: empty secret must never disable webhook authentication.
	if v.secret == "" {
		return ErrUnauthorized
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

func parseJSONWebhook[T any](v tokenValidator, headers http.Header, body []byte, toEvent func(T) (PortalEvent, bool)) (PortalEvent, error) {
	if err := v.validate(headers); err != nil {
		return PortalEvent{}, err
	}
	var payload T
	if err := json.Unmarshal(body, &payload); err != nil {
		return PortalEvent{}, ErrInvalidPayload
	}
	ev, ok := toEvent(payload)
	if !ok || ev.RoomNumber == "" || ev.GuestIP == "" {
		return PortalEvent{}, ErrInvalidPayload
	}
	return ev, nil
}

// MikroTikAdapter handles MikroTik Hotspot login webhooks.
type MikroTikAdapter struct {
	tokenValidator
}

func NewMikroTikAdapter(secret string) *MikroTikAdapter {
	return &MikroTikAdapter{tokenValidator: tokenValidator{secret: secret}}
}

func (a *MikroTikAdapter) Vendor() string { return "mikrotik" }

type mikrotikPayload struct {
	Room      string `json:"room"`
	GuestName string `json:"guest_name"`
	IP        string `json:"ip"`
	MAC       string `json:"mac"`
}

func (a *MikroTikAdapter) ParseWebhook(headers http.Header, body []byte) (PortalEvent, error) {
	return parseJSONWebhook(a.tokenValidator, headers, body, func(p mikrotikPayload) (PortalEvent, bool) {
		return PortalEvent{
			RoomNumber: p.Room,
			GuestName:  p.GuestName,
			GuestIP:    p.IP,
			GuestMAC:   p.MAC,
		}, true
	})
}

// UniFiAdapter handles Ubiquiti UniFi Network external portal webhooks.
type UniFiAdapter struct {
	tokenValidator
}

func NewUniFiAdapter(secret string) *UniFiAdapter {
	return &UniFiAdapter{tokenValidator: tokenValidator{secret: secret}}
}

func (a *UniFiAdapter) Vendor() string { return "unifi" }

type unifiPayload struct {
	RoomNumber string `json:"room_number"`
	GuestName  string `json:"guest_name"`
	ClientIP   string `json:"client_ip"`
	ClientMAC  string `json:"client_mac"`
}

func (a *UniFiAdapter) ParseWebhook(headers http.Header, body []byte) (PortalEvent, error) {
	return parseJSONWebhook(a.tokenValidator, headers, body, func(p unifiPayload) (PortalEvent, bool) {
		return PortalEvent{
			RoomNumber: p.RoomNumber,
			GuestName:  p.GuestName,
			GuestIP:    p.ClientIP,
			GuestMAC:   p.ClientMAC,
		}, true
	})
}

// RuijieAdapter handles Ruijie Cloud/Gateway captive portal webhooks.
type RuijieAdapter struct {
	tokenValidator
}

func NewRuijieAdapter(secret string) *RuijieAdapter {
	return &RuijieAdapter{tokenValidator: tokenValidator{secret: secret}}
}

func (a *RuijieAdapter) Vendor() string { return "ruijie" }

type ruijiePayload struct {
	RoomNo    string `json:"roomNo"`
	GuestName string `json:"guestName"`
	UserIP    string `json:"userIp"`
	UserMAC   string `json:"userMac"`
}

func (a *RuijieAdapter) ParseWebhook(headers http.Header, body []byte) (PortalEvent, error) {
	return parseJSONWebhook(a.tokenValidator, headers, body, func(p ruijiePayload) (PortalEvent, bool) {
		return PortalEvent{
			RoomNumber: p.RoomNo,
			GuestName:  p.GuestName,
			GuestIP:    p.UserIP,
			GuestMAC:   p.UserMAC,
		}, true
	})
}

// OmadaAdapter handles TP-Link Omada Controller external portal webhooks.
type OmadaAdapter struct {
	tokenValidator
}

func NewOmadaAdapter(secret string) *OmadaAdapter {
	return &OmadaAdapter{tokenValidator: tokenValidator{secret: secret}}
}

func (a *OmadaAdapter) Vendor() string { return "omada" }

type omadaPayload struct {
	Room      string `json:"room"`
	Name      string `json:"name"`
	ClientIP  string `json:"clientIp"`
	ClientMAC string `json:"clientMac"`
}

func (a *OmadaAdapter) ParseWebhook(headers http.Header, body []byte) (PortalEvent, error) {
	return parseJSONWebhook(a.tokenValidator, headers, body, func(p omadaPayload) (PortalEvent, bool) {
		return PortalEvent{
			RoomNumber: p.Room,
			GuestName:  p.Name,
			GuestIP:    p.ClientIP,
			GuestMAC:   p.ClientMAC,
		}, true
	})
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

		body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodyBytes+1))
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		if len(body) > maxWebhookBodyBytes {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}

		ev, err := adapter.ParseWebhook(r.Header, body)
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
