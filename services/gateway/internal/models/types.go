package models

import "time"

type PairingMode string

const (
	PairModePortal   PairingMode = "portal"
	PairModeRADIUS   PairingMode = "radius"
	PairModeQRPIN    PairingMode = "qr_pin"
	PairModeFOManual PairingMode = "fo_manual"
)

type GuestSegment string

const (
	SegmentLeisure   GuestSegment = "leisure"
	SegmentCorporate GuestSegment = "corporate"
	SegmentVIP       GuestSegment = "vip"
)

type MessagePriority string

const (
	PriorityNormal    MessagePriority = "normal"
	PriorityUrgent    MessagePriority = "urgent"
	PriorityEmergency MessagePriority = "emergency"
)

type ReceiverDevice struct {
	DeviceID       string    `json:"device_id"`
	RoomNumber     string    `json:"room_number"`
	Label          string    `json:"label"`
	IP             string    `json:"ip"`
	MAC            string    `json:"mac,omitempty"`
	CastPort       int       `json:"cast_port"`
	AirPlayEnabled bool      `json:"airplay_enabled"`
	AirPlayPort    int       `json:"airplay_port"`
	ModelName      string    `json:"model_name"`
	AppVersion     string    `json:"app_version,omitempty"`
	HDMICEC        bool      `json:"hdmi_cec_attached"`
	Mode           string    `json:"mode,omitempty"`        // "room_tv" | "signage"
	Orientation    string    `json:"orientation,omitempty"` // "landscape" | "portrait"
	Status         string    `json:"status"`
	LastSeenAt     time.Time `json:"last_seen_at"`
}

type Room struct {
	RoomNumber  string       `json:"room_number"`
	Floor       string       `json:"floor"`
	Segment     GuestSegment `json:"segment"`
	Occupied    bool         `json:"occupied"`
	GuestName   string       `json:"guest_name,omitempty"`
	CheckedInAt time.Time    `json:"checked_in_at,omitempty"`
	CheckoutAt  time.Time    `json:"checkout_at,omitempty"`
	ResetEpoch  int64        `json:"reset_epoch"`
	CurrentPIN  string       `json:"current_pin"`
	PINExpires  time.Time    `json:"pin_expires"`
}

type GuestSession struct {
	SessionID       string      `json:"session_id"`
	RadiusSessionID string      `json:"radius_session_id,omitempty"`
	RoomNumber      string      `json:"room_number"`
	GuestIP         string      `json:"guest_ip"`
	GuestMAC        string      `json:"guest_mac,omitempty"`
	GuestName       string      `json:"guest_name,omitempty"`
	PairingMode     PairingMode `json:"pairing_mode"`
	CreatedAt       time.Time   `json:"created_at"`
	ExpiresAt       time.Time   `json:"expires_at"`
}

type RadiusAccountingEvent struct {
	StatusType      string `json:"acct_status_type"` // "Start", "Interim-Update", "Stop"
	RadiusSessionID string `json:"acct_session_id"`
	RoomNumber      string `json:"room_number,omitempty"`
	GuestIP         string `json:"framed_ip_address"`
	GuestMAC        string `json:"calling_station_id,omitempty"`
}

type AdCampaign struct {
	ID             string         `json:"id"`
	Title          string         `json:"title"`
	MediaURL       string         `json:"media_url"`
	CTAURL         string         `json:"cta_url,omitempty"`
	TargetSegments []GuestSegment `json:"target_segments"`
	DurationSec    int            `json:"duration_sec"`
	Active         bool           `json:"active"`
}

type GuestMessage struct {
	ID            string          `json:"id"`
	RoomNumber    string          `json:"room_number,omitempty"` // empty = property broadcast
	Title         string          `json:"title"`
	Body          string          `json:"body"`
	Priority      MessagePriority `json:"priority"`
	ShowAsOverlay bool            `json:"show_as_overlay"`
	CreatedAt     time.Time       `json:"created_at"`
}

type PropertyConfig struct {
	HotelName       string `json:"hotel_name"`
	WiFiSSID        string `json:"wifi_ssid"`
	WelcomeTitle    string `json:"welcome_title"`
	WelcomeSubtitle string `json:"welcome_subtitle"`
	BasePairingURL  string `json:"base_pairing_url"`
	MaxDevicesRoom  int    `json:"max_devices_per_room"`
}

type EdgeSnapshot struct {
	Config    PropertyConfig            `json:"config"`
	Rooms     map[string]*Room          `json:"rooms"`
	Receivers map[string]*ReceiverDevice `json:"receivers"`
	Sessions  map[string]*GuestSession  `json:"sessions"` // keyed by guest_ip
	Ads       []AdCampaign              `json:"ads"`
	Messages  []GuestMessage            `json:"messages"`
	SavedAt   time.Time                 `json:"saved_at"`
}

type PairReceiverInfo struct {
	DeviceID string `json:"device_id"`
	Label    string `json:"label"`
	IP       string `json:"ip"`
}

type PairResult struct {
	SessionID   string             `json:"session_id"`
	RoomNumber  string             `json:"room_number"`
	GuestIP     string             `json:"guest_ip"`
	PairingMode PairingMode        `json:"pairing_mode"`
	Receivers   []PairReceiverInfo `json:"receivers"`
	ExpiresAt   time.Time          `json:"expires_at"`
}

type TVAdView struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	MediaURL    string `json:"media_url"`
	CTAURL      string `json:"cta_url,omitempty"`
	DurationSec int    `json:"duration_sec"`
}

type TVMessageView struct {
	ID            string          `json:"id"`
	Title         string          `json:"title"`
	Body          string          `json:"body"`
	Priority      MessagePriority `json:"priority"`
	ShowAsOverlay bool            `json:"show_as_overlay"`
}

type TVBackdropState struct {
	RoomNumber          string          `json:"room_number"`
	DeviceID            string          `json:"device_id"`
	DeviceStatus        string          `json:"device_status,omitempty"`
	DeviceLabel         string          `json:"device_label"`
	Mode                string          `json:"mode,omitempty"`
	Orientation         string          `json:"orientation,omitempty"`
	Occupied            bool            `json:"occupied"`
	GuestName           string          `json:"guest_name,omitempty"`
	Segment             GuestSegment    `json:"segment"`
	HotelName           string          `json:"hotel_name"`
	WelcomeTitle        string          `json:"welcome_title"`
	WelcomeSubtitle     string          `json:"welcome_subtitle"`
	WiFiSSID            string          `json:"wifi_ssid"`
	PairingPIN          string          `json:"pairing_pin"`
	PairingURL          string          `json:"pairing_url"`
	ActivePairedDevices int             `json:"active_paired_devices"`
	ResetEpoch          int64           `json:"reset_epoch"`
	Ads                 []TVAdView      `json:"ads"`
	Messages            []TVMessageView `json:"messages"`
}
