package session

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/inaply/gateway/internal/firewall"
	"github.com/inaply/gateway/internal/models"
)

var (
	ErrRoomNotFound           = errors.New("room not found")
	ErrSignagePairingForbidden = errors.New("guest pairing is not allowed on signage devices")
	ErrDeviceNotFound    = errors.New("receiver device not found")
	ErrInvalidPIN        = errors.New("invalid or expired pairing PIN")
	ErrGuestMismatch     = errors.New("guest name does not match room check-in record")
	ErrInvalidGuestIP    = errors.New("invalid guest IP address")
	ErrRoomNotConfigured = errors.New("room has no registered receiver devices")
)

type PersistHook func(models.EdgeSnapshot) error

// Store manages room state, receiver registry, hybrid guest pairing sessions,
// TV ads/messages, and dynamic firewall synchronization.
type Store struct {
	mu        sync.RWMutex
	config    models.PropertyConfig
	rooms     map[string]*models.Room
	receivers map[string]*models.ReceiverDevice
	sessions  map[string]*models.GuestSession // keyed by guest IP
	ads       []models.AdCampaign
	messages  []models.GuestMessage
	fw        firewall.Driver
	persist   PersistHook
	nowFn     func() time.Time
}

func NewStore(fw firewall.Driver, persist PersistHook) *Store {
	if fw == nil {
		fw = firewall.NewMemoryDriver()
	}
	return &Store{
		config: models.PropertyConfig{
			HotelName:       "Nusantara Grand Hotel",
			WiFiSSID:        "Nusantara-Guest",
			WelcomeTitle:    "Welcome to Nusantara Grand Hotel",
			WelcomeSubtitle: "Stay on hotel Wi-Fi and tap Cast in your favorite app — no QR or PIN needed.",
			BasePairingURL:  "http://localhost:8080/pair",
			MaxDevicesRoom:  5,
		},
		rooms:     make(map[string]*models.Room),
		receivers: make(map[string]*models.ReceiverDevice),
		sessions:  make(map[string]*models.GuestSession),
		fw:        fw,
		persist:   persist,
		nowFn:     time.Now,
	}
}

func (s *Store) SetNowFunc(fn func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nowFn = fn
}

func (s *Store) SeedDemoProperty() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.nowFn()
	defaultOut := time.Date(now.Year(), now.Month(), now.Day()+1, 12, 0, 0, 0, now.Location())

	s.rooms["301"] = &models.Room{
		RoomNumber:  "301",
		Floor:       "3",
		Segment:     models.SegmentLeisure,
		Occupied:    true,
		GuestName:   "Budi Santoso",
		CheckedInAt: now.Add(-2 * time.Hour),
		CheckoutAt:  defaultOut,
		ResetEpoch:  1,
		CurrentPIN:  "301842",
		PINExpires:  now.Add(24 * time.Hour),
	}
	s.rooms["302"] = &models.Room{
		RoomNumber:  "302",
		Floor:       "3",
		Segment:     models.SegmentCorporate,
		Occupied:    true,
		GuestName:   "Siti Rahma",
		CheckedInAt: now.Add(-4 * time.Hour),
		CheckoutAt:  defaultOut,
		ResetEpoch:  1,
		CurrentPIN:  "302915",
		PINExpires:  now.Add(24 * time.Hour),
	}
	s.rooms["501"] = &models.Room{
		RoomNumber:  "501",
		Floor:       "5",
		Segment:     models.SegmentVIP,
		Occupied:    true,
		GuestName:   "Andi Wijaya",
		CheckedInAt: now.Add(-1 * time.Hour),
		CheckoutAt:  defaultOut,
		ResetEpoch:  1,
		CurrentPIN:  "501774",
		PINExpires:  now.Add(24 * time.Hour),
	}

	s.receivers["tv-301-main"] = &models.ReceiverDevice{
		DeviceID:       "tv-301-main",
		RoomNumber:     "301",
		Label:          "Room 301 TV",
		IP:             "10.20.3.101",
		CastPort:       8009,
		AirPlayEnabled: true,
		AirPlayPort:    7000,
		ModelName:      "Google TV Streamer",
		Status:         "idle",
		HDMICEC:        true,
		LastSeenAt:     now,
	}
	s.receivers["tv-302-main"] = &models.ReceiverDevice{
		DeviceID:       "tv-302-main",
		RoomNumber:     "302",
		Label:          "Room 302 TV",
		IP:             "10.20.3.102",
		CastPort:       8009,
		AirPlayEnabled: true,
		AirPlayPort:    7000,
		ModelName:      "Google TV Streamer",
		Status:         "idle",
		HDMICEC:        true,
		LastSeenAt:     now,
	}
	s.receivers["tv-501-living"] = &models.ReceiverDevice{
		DeviceID:       "tv-501-living",
		RoomNumber:     "501",
		Label:          "Suite 501 Living Room",
		IP:             "10.20.5.101",
		CastPort:       8009,
		AirPlayEnabled: true,
		AirPlayPort:    7000,
		ModelName:      "Google TV Streamer",
		Status:         "idle",
		HDMICEC:        true,
		LastSeenAt:     now,
	}
	s.receivers["tv-501-bedroom"] = &models.ReceiverDevice{
		DeviceID:       "tv-501-bedroom",
		RoomNumber:     "501",
		Label:          "Suite 501 Master Bedroom",
		IP:             "10.20.5.102",
		CastPort:       8009,
		AirPlayEnabled: true,
		AirPlayPort:    7000,
		ModelName:      "Google TV Streamer",
		Status:         "idle",
		HDMICEC:        true,
		LastSeenAt:     now,
	}

	s.ads = []models.AdCampaign{
		{
			ID:             "ad-spa-20",
			Title:          "Diskon 20% Balinese Spa Hari Ini",
			MediaURL:       "/assets/ads/spa-offer.jpg",
			CTAURL:         "https://hotel.example/spa",
			TargetSegments: []models.GuestSegment{models.SegmentLeisure, models.SegmentVIP},
			DurationSec:    15,
			Active:         true,
		},
		{
			ID:             "ad-dining-dinner",
			Title:          "Sunset Seafood BBQ di Rooftop Bistro",
			MediaURL:       "/assets/ads/rooftop-bbq.jpg",
			CTAURL:         "https://hotel.example/dining",
			TargetSegments: []models.GuestSegment{models.SegmentLeisure, models.SegmentCorporate, models.SegmentVIP},
			DurationSec:    15,
			Active:         true,
		},
		{
			ID:             "ad-meeting-lounge",
			Title:          "Executive Lounge & Paket Meeting Cepat",
			MediaURL:       "/assets/ads/exec-lounge.jpg",
			CTAURL:         "https://hotel.example/business",
			TargetSegments: []models.GuestSegment{models.SegmentCorporate},
			DurationSec:    15,
			Active:         true,
		},
	}

	s.messages = []models.GuestMessage{
		{
			ID:            "msg-welcome",
			RoomNumber:    "",
			Title:         "Selamat Datang",
			Body:          "Nikmati streaming aplikasi pribadi Anda langsung ke TV kamar tanpa login di TV.",
			Priority:      models.PriorityNormal,
			ShowAsOverlay: false,
			CreatedAt:     now,
		},
	}

	_ = s.persistLocked()
}

func (s *Store) LoadSnapshot(snap models.EdgeSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = snap.Config
	s.rooms = snap.Rooms
	s.receivers = snap.Receivers
	s.sessions = snap.Sessions
	s.ads = snap.Ads
	s.messages = snap.Messages

	// Re-apply active firewall rules on startup from persisted edge sessions.
	now := s.nowFn()
	for ip, sess := range s.sessions {
		if now.After(sess.ExpiresAt) {
			delete(s.sessions, ip)
			continue
		}
		for _, rcv := range s.receiversForRoomLocked(sess.RoomNumber) {
			if err := s.fw.AllowPair(sess.GuestIP, rcv.IP); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) Snapshot() models.EdgeSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshotLocked()
}

func (s *Store) snapshotLocked() models.EdgeSnapshot {
	rooms := make(map[string]*models.Room, len(s.rooms))
	for k, v := range s.rooms {
		cp := *v
		rooms[k] = &cp
	}
	receivers := make(map[string]*models.ReceiverDevice, len(s.receivers))
	for k, v := range s.receivers {
		cp := *v
		receivers[k] = &cp
	}
	sessions := make(map[string]*models.GuestSession, len(s.sessions))
	for k, v := range s.sessions {
		cp := *v
		sessions[k] = &cp
	}
	ads := append([]models.AdCampaign(nil), s.ads...)
	msgs := append([]models.GuestMessage(nil), s.messages...)

	return models.EdgeSnapshot{
		Config:    s.config,
		Rooms:     rooms,
		Receivers: receivers,
		Sessions:  sessions,
		Ads:       ads,
		Messages:  msgs,
		SavedAt:   s.nowFn(),
	}
}

func (s *Store) persistLocked() error {
	if s.persist == nil {
		return nil
	}
	return s.persist(s.snapshotLocked())
}

func (s *Store) UpsertRoom(room models.Room) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copyRoom := room
	s.rooms[room.RoomNumber] = &copyRoom
}

func (s *Store) UpsertReceiver(rcv models.ReceiverDevice) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copyRcv := rcv
	s.receivers[rcv.DeviceID] = &copyRcv
}

func (s *Store) receiversForRoomLocked(roomNumber string) []models.ReceiverDevice {
	var list []models.ReceiverDevice
	for _, r := range s.receivers {
		if r.RoomNumber == roomNumber {
			list = append(list, *r)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].DeviceID < list[j].DeviceID
	})
	return list
}

func (s *Store) ReceiversForGuestIP(guestIP string) (*models.Room, []models.ReceiverDevice) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.sessions[guestIP]
	if !ok {
		return nil, nil
	}
	if s.nowFn().After(sess.ExpiresAt) {
		return nil, nil
	}
	room, ok := s.rooms[sess.RoomNumber]
	if !ok {
		return nil, nil
	}
	cp := *room
	return &cp, s.receiversForRoomLocked(sess.RoomNumber)
}

func (s *Store) CheckIn(roomNumber, guestName string, segment models.GuestSegment, checkoutAt time.Time) (*models.Room, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.rooms[roomNumber]
	if !ok {
		return nil, ErrRoomNotFound
	}
	now := s.nowFn()
	if checkoutAt.IsZero() {
		checkoutAt = time.Date(now.Year(), now.Month(), now.Day()+1, 12, 0, 0, 0, now.Location())
	}
	if segment == "" {
		segment = models.SegmentLeisure
	}
	// Clear any leftover sessions from previous stay before checking in new guest.
	if err := s.revokeRoomSessionsLocked(roomNumber); err != nil {
		return nil, err
	}

	room.Occupied = true
	room.GuestName = strings.TrimSpace(guestName)
	room.Segment = segment
	room.CheckedInAt = now
	room.CheckoutAt = checkoutAt
	room.ResetEpoch++
	room.CurrentPIN = generatePIN()
	room.PINExpires = now.Add(24 * time.Hour)

	_ = s.persistLocked()
	cp := *room
	return &cp, nil
}

func (s *Store) CheckOut(roomNumber string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.rooms[roomNumber]
	if !ok {
		return 0, ErrRoomNotFound
	}
	removed := s.countRoomSessionsLocked(roomNumber)
	if err := s.revokeRoomSessionsLocked(roomNumber); err != nil {
		return 0, err
	}

	// Remove room-targeted messages upon checkout so next guest does not read them.
	filtered := s.messages[:0]
	for _, m := range s.messages {
		if m.RoomNumber != roomNumber {
			filtered = append(filtered, m)
		}
	}
	s.messages = filtered

	now := s.nowFn()
	room.Occupied = false
	room.GuestName = ""
	room.CheckoutAt = time.Time{}
	room.ResetEpoch++
	room.CurrentPIN = generatePIN()
	room.PINExpires = now.Add(24 * time.Hour)

	_ = s.persistLocked()
	return removed, nil
}

func (s *Store) PairViaPortal(roomNumber, guestName, guestIP, guestMAC string) (*models.PairResult, error) {
	return s.pairRoom(roomNumber, guestName, guestIP, guestMAC, models.PairModePortal, true)
}

func (s *Store) PairViaManualFO(roomNumber, guestIP, guestMAC string) (*models.PairResult, error) {
	return s.pairRoom(roomNumber, "", guestIP, guestMAC, models.PairModeFOManual, false)
}

func (s *Store) PairViaPIN(pin, guestIP, guestMAC string) (*models.PairResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pin = strings.TrimSpace(pin)
	now := s.nowFn()
	var matchedRoom *models.Room
	for _, r := range s.rooms {
		if r.CurrentPIN == pin && now.Before(r.PINExpires) {
			matchedRoom = r
			break
		}
	}
	if matchedRoom == nil {
		return nil, ErrInvalidPIN
	}
	return s.bindGuestLocked(matchedRoom, guestIP, guestMAC, matchedRoom.GuestName, models.PairModeQRPIN)
}

func (s *Store) pairRoom(roomNumber, guestName, guestIP, guestMAC string, mode models.PairingMode, verifyName bool) (*models.PairResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.rooms[roomNumber]
	if !ok {
		return nil, ErrRoomNotFound
	}
	if verifyName && room.Occupied && room.GuestName != "" {
		if !guestMatches(room.GuestName, guestName) {
			return nil, ErrGuestMismatch
		}
	}
	if !room.Occupied && strings.TrimSpace(guestName) != "" {
		// Allow self-registration during demo/unintegrated hotel mode when unoccupied.
		room.Occupied = true
		room.GuestName = strings.TrimSpace(guestName)
		room.CheckedInAt = s.nowFn()
	}
	return s.bindGuestLocked(room, guestIP, guestMAC, room.GuestName, mode)
}

func (s *Store) bindGuestLocked(room *models.Room, guestIP, guestMAC, guestName string, mode models.PairingMode) (*models.PairResult, error) {
	if net.ParseIP(guestIP) == nil {
		return nil, ErrInvalidGuestIP
	}
	receivers := s.receiversForRoomLocked(room.RoomNumber)
	if len(receivers) == 0 {
		return nil, ErrRoomNotConfigured
	}
	for _, rcv := range receivers {
		if rcv.Mode == "signage" {
			return nil, ErrSignagePairingForbidden
		}
	}

	// If this guest IP was previously bound to another room, revoke old room firewall rules first.
	if existing, ok := s.sessions[guestIP]; ok && existing.RoomNumber != room.RoomNumber {
		for _, oldRcv := range s.receiversForRoomLocked(existing.RoomNumber) {
			_ = s.fw.RevokePair(guestIP, oldRcv.IP)
		}
		delete(s.sessions, guestIP)
	}

	// Enforce max devices per room (evicting oldest device if exceeded).
	s.enforceMaxDevicesLocked(room.RoomNumber, guestIP)

	now := s.nowFn()
	expiresAt := room.CheckoutAt
	if expiresAt.IsZero() || expiresAt.Before(now) {
		expiresAt = now.Add(24 * time.Hour)
	}

	for _, rcv := range receivers {
		if err := s.fw.AllowPair(guestIP, rcv.IP); err != nil {
			return nil, err
		}
	}

	sess := &models.GuestSession{
		SessionID:   randomHex(8),
		RoomNumber:  room.RoomNumber,
		GuestIP:     guestIP,
		GuestMAC:    guestMAC,
		GuestName:   guestName,
		PairingMode: mode,
		CreatedAt:   now,
		ExpiresAt:   expiresAt,
	}
	s.sessions[guestIP] = sess
	_ = s.persistLocked()

	resReceivers := make([]models.PairReceiverInfo, 0, len(receivers))
	for _, r := range receivers {
		resReceivers = append(resReceivers, models.PairReceiverInfo{
			DeviceID: r.DeviceID,
			Label:    r.Label,
			IP:       r.IP,
		})
	}
	return &models.PairResult{
		SessionID:   sess.SessionID,
		RoomNumber:  room.RoomNumber,
		GuestIP:     guestIP,
		PairingMode: mode,
		Receivers:   resReceivers,
		ExpiresAt:   expiresAt,
	}, nil
}

func (s *Store) enforceMaxDevicesLocked(roomNumber, incomingIP string) {
	limit := s.config.MaxDevicesRoom
	if limit <= 0 {
		return
	}
	var roomSessions []*models.GuestSession
	for _, sess := range s.sessions {
		if sess.RoomNumber == roomNumber && sess.GuestIP != incomingIP {
			roomSessions = append(roomSessions, sess)
		}
	}
	if len(roomSessions) < limit {
		return
	}
	sort.Slice(roomSessions, func(i, j int) bool {
		return roomSessions[i].CreatedAt.Before(roomSessions[j].CreatedAt)
	})
	toEvict := len(roomSessions) - limit + 1
	for i := 0; i < toEvict; i++ {
		ev := roomSessions[i]
		for _, rcv := range s.receiversForRoomLocked(roomNumber) {
			_ = s.fw.RevokePair(ev.GuestIP, rcv.IP)
		}
		delete(s.sessions, ev.GuestIP)
	}
}

func (s *Store) revokeRoomSessionsLocked(roomNumber string) error {
	receivers := s.receiversForRoomLocked(roomNumber)
	for ip, sess := range s.sessions {
		if sess.RoomNumber == roomNumber {
			for _, rcv := range receivers {
				if err := s.fw.RevokePair(sess.GuestIP, rcv.IP); err != nil {
					return err
				}
			}
			delete(s.sessions, ip)
		}
	}
	return nil
}

func (s *Store) countRoomSessionsLocked(roomNumber string) int {
	n := 0
	for _, sess := range s.sessions {
		if sess.RoomNumber == roomNumber {
			n++
		}
	}
	return n
}

func (s *Store) TVState(deviceID string) (*models.TVBackdropState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rcv, ok := s.receivers[deviceID]
	if !ok {
		return nil, ErrDeviceNotFound
	}
	if rcv.Mode == "signage" && rcv.RoomNumber == "" {
		ads := make([]models.TVAdView, 0)
		for _, ad := range s.ads {
			if !ad.Active {
				continue
			}
			ads = append(ads, models.TVAdView{
				ID:          ad.ID,
				Title:       ad.Title,
				MediaURL:    ad.MediaURL,
				CTAURL:      ad.CTAURL,
				DurationSec: ad.DurationSec,
			})
		}
		msgs := make([]models.TVMessageView, 0)
		for _, m := range s.messages {
			if m.RoomNumber == "" {
				msgs = append(msgs, models.TVMessageView{
					ID:            m.ID,
					Title:         m.Title,
					Body:          m.Body,
					Priority:      m.Priority,
					ShowAsOverlay: m.ShowAsOverlay,
				})
			}
		}
		return &models.TVBackdropState{
			DeviceID:        rcv.DeviceID,
			DeviceStatus:    rcv.Status,
			DeviceLabel:     rcv.Label,
			Mode:            rcv.Mode,
			Orientation:     rcv.Orientation,
			HotelName:       s.config.HotelName,
			WelcomeTitle:    s.config.WelcomeTitle,
			WelcomeSubtitle: s.config.WelcomeSubtitle,
			Ads:             ads,
			Messages:        msgs,
		}, nil
	}
	room, ok := s.rooms[rcv.RoomNumber]
	if !ok {
		return nil, ErrRoomNotFound
	}

	ads := make([]models.TVAdView, 0)
	for _, ad := range s.ads {
		if !ad.Active || !segmentAllowed(ad.TargetSegments, room.Segment) {
			continue
		}
		ads = append(ads, models.TVAdView{
			ID:          ad.ID,
			Title:       ad.Title,
			MediaURL:    ad.MediaURL,
			CTAURL:      ad.CTAURL,
			DurationSec: ad.DurationSec,
		})
	}

	msgs := make([]models.TVMessageView, 0)
	for _, m := range s.messages {
		if m.RoomNumber == "" || m.RoomNumber == room.RoomNumber {
			msgs = append(msgs, models.TVMessageView{
				ID:            m.ID,
				Title:         m.Title,
				Body:          m.Body,
				Priority:      m.Priority,
				ShowAsOverlay: m.ShowAsOverlay,
			})
		}
	}

	welcomeTitle := s.config.WelcomeTitle
	if room.Occupied && room.GuestName != "" {
		welcomeTitle = fmt.Sprintf("Welcome, %s", room.GuestName)
	}

	return &models.TVBackdropState{
		RoomNumber:          room.RoomNumber,
		DeviceID:            rcv.DeviceID,
		DeviceStatus:        rcv.Status,
		DeviceLabel:         rcv.Label,
		Mode:                rcv.Mode,
		Orientation:         rcv.Orientation,
		Occupied:            room.Occupied,
		GuestName:           room.GuestName,
		Segment:             room.Segment,
		HotelName:           s.config.HotelName,
		WelcomeTitle:        welcomeTitle,
		WelcomeSubtitle:     s.config.WelcomeSubtitle,
		WiFiSSID:            s.config.WiFiSSID,
		PairingPIN:          room.CurrentPIN,
		PairingURL:          fmt.Sprintf("%s?pin=%s", s.config.BasePairingURL, room.CurrentPIN),
		ActivePairedDevices: s.countRoomSessionsLocked(room.RoomNumber),
		ResetEpoch:          room.ResetEpoch,
		Ads:                 ads,
		Messages:            msgs,
	}, nil
}

func (s *Store) Heartbeat(deviceID, status, appVersion string, cecAttached bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rcv, ok := s.receivers[deviceID]
	if !ok {
		return ErrDeviceNotFound
	}
	if status != "" {
		rcv.Status = status
	}
	if appVersion != "" {
		rcv.AppVersion = appVersion
	}
	rcv.HDMICEC = cecAttached
	rcv.LastSeenAt = s.nowFn()
	return nil
}

func (s *Store) AddMessage(msg models.GuestMessage) models.GuestMessage {
	s.mu.Lock()
	defer s.mu.Unlock()

	if msg.ID == "" {
		msg.ID = "msg-" + randomHex(4)
	}
	if msg.Priority == "" {
		msg.Priority = models.PriorityNormal
	}
	msg.CreatedAt = s.nowFn()
	s.messages = append(s.messages, msg)
	_ = s.persistLocked()
	return msg
}

func (s *Store) AddAd(ad models.AdCampaign) models.AdCampaign {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ad.ID == "" {
		ad.ID = "ad-" + randomHex(4)
	}
	if ad.DurationSec <= 0 {
		ad.DurationSec = 15
	}
	if len(ad.TargetSegments) == 0 {
		ad.TargetSegments = []models.GuestSegment{models.SegmentLeisure, models.SegmentCorporate, models.SegmentVIP}
	}
	ad.Active = true
	s.ads = append(s.ads, ad)
	_ = s.persistLocked()
	return ad
}

func (s *Store) ExpireStale(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	expiredCount := 0
	for ip, sess := range s.sessions {
		if now.After(sess.ExpiresAt) {
			for _, rcv := range s.receiversForRoomLocked(sess.RoomNumber) {
				_ = s.fw.RevokePair(sess.GuestIP, rcv.IP)
			}
			delete(s.sessions, ip)
			expiredCount++
		}
	}
	for _, room := range s.rooms {
		if now.After(room.PINExpires) {
			room.CurrentPIN = generatePIN()
			room.PINExpires = now.Add(24 * time.Hour)
		}
	}
	if expiredCount > 0 {
		_ = s.persistLocked()
	}
	return expiredCount
}

func (s *Store) ListRoomsAndSessions() ([]models.Room, []models.GuestSession, []models.ReceiverDevice) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rooms := make([]models.Room, 0, len(s.rooms))
	for _, r := range s.rooms {
		rooms = append(rooms, *r)
	}
	sort.Slice(rooms, func(i, j int) bool { return rooms[i].RoomNumber < rooms[j].RoomNumber })

	sessions := make([]models.GuestSession, 0, len(s.sessions))
	for _, sess := range s.sessions {
		sessions = append(sessions, *sess)
	}
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].CreatedAt.Before(sessions[j].CreatedAt) })

	receivers := make([]models.ReceiverDevice, 0, len(s.receivers))
	for _, rcv := range s.receivers {
		receivers = append(receivers, *rcv)
	}
	sort.Slice(receivers, func(i, j int) bool { return receivers[i].DeviceID < receivers[j].DeviceID })

	return rooms, sessions, receivers
}

// DisconnectGuestIP revokes only the session and firewall pairs belonging to guestIP.
func (s *Store) DisconnectGuestIP(guestIP string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[guestIP]
	if !ok {
		return nil
	}
	for _, rcv := range s.receiversForRoomLocked(sess.RoomNumber) {
		_ = s.fw.RevokePair(sess.GuestIP, rcv.IP)
	}
	delete(s.sessions, guestIP)
	return s.persistLocked()
}

// HandleRadiusAccounting synchronizes guest IP/MAC changes (roaming & MAC randomization)
// and session termination from RADIUS Accounting Start/Interim-Update/Stop events.
func (s *Store) HandleRadiusAccounting(ev models.RadiusAccountingEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess := s.findSessionForRadiusLocked(ev)
	if sess == nil {
		return nil
	}

	receivers := s.receiversForRoomLocked(sess.RoomNumber)
	if ev.StatusType == "Stop" {
		for _, rcv := range receivers {
			_ = s.fw.RevokePair(sess.GuestIP, rcv.IP)
		}
		delete(s.sessions, sess.GuestIP)
		return s.persistLocked()
	}

	// Start or Interim-Update: handle IP roaming and MAC randomization.
	if ev.GuestIP != "" && ev.GuestIP != sess.GuestIP {
		for _, rcv := range receivers {
			_ = s.fw.RevokePair(sess.GuestIP, rcv.IP)
		}
		delete(s.sessions, sess.GuestIP)
		sess.GuestIP = ev.GuestIP
		s.sessions[sess.GuestIP] = sess
		for _, rcv := range receivers {
			if err := s.fw.AllowPair(sess.GuestIP, rcv.IP); err != nil {
				return err
			}
		}
	}
	if ev.GuestMAC != "" {
		sess.GuestMAC = ev.GuestMAC
	}
	if ev.RadiusSessionID != "" {
		sess.RadiusSessionID = ev.RadiusSessionID
	}
	return s.persistLocked()
}

func (s *Store) findSessionForRadiusLocked(ev models.RadiusAccountingEvent) *models.GuestSession {
	if ev.RadiusSessionID != "" {
		for _, sess := range s.sessions {
			if sess.RadiusSessionID == ev.RadiusSessionID {
				return sess
			}
		}
	}
	if ev.GuestMAC != "" {
		for _, sess := range s.sessions {
			if sess.GuestMAC == ev.GuestMAC {
				return sess
			}
		}
	}
	if ev.GuestIP != "" {
		if sess, ok := s.sessions[ev.GuestIP]; ok {
			return sess
		}
	}
	return nil
}

func segmentAllowed(targets []models.GuestSegment, current models.GuestSegment) bool {
	if len(targets) == 0 {
		return true
	}
	for _, t := range targets {
		if t == current {
			return true
		}
	}
	return false
}

func guestMatches(expected, provided string) bool {
	exp := strings.ToLower(strings.TrimSpace(expected))
	prov := strings.ToLower(strings.TrimSpace(provided))
	if prov == "" {
		return false
	}
	// Accept full name or last-name/first-name match as hotels commonly require surname only.
	if exp == prov {
		return true
	}
	for _, part := range strings.Fields(exp) {
		if part == prov {
			return true
		}
	}
	return false
}

func generatePIN() string {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "482910"
	}
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

func randomHex(numBytes int) string {
	b := make([]byte, numBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
