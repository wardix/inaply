package session

import (
	"strings"
	"testing"
	"time"

	"github.com/inaply/gateway/internal/firewall"
	"github.com/inaply/gateway/internal/models"
)

func TestSessionLifecycleAndIsolation(t *testing.T) {
	fw := firewall.NewMemoryDriver()
	store := NewStore(fw, nil)
	baseNow := time.Date(2026, 11, 26, 14, 0, 0, 0, time.UTC)
	store.SetNowFunc(func() time.Time { return baseNow })
	store.SeedDemoProperty()

	// Reject mismatched guest surname on portal pairing
	if _, err := store.PairViaPortal("301", "WrongName", "10.10.30.5", ""); err != ErrGuestMismatch {
		t.Fatalf("expected ErrGuestMismatch, got %v", err)
	}

	// Pair guest to Room 301 via surname
	res, err := store.PairViaPortal("301", "budi", "10.10.30.5", "aa:bb:cc:dd:ee:01")
	if err != nil {
		t.Fatalf("PairViaPortal failed: %v", err)
	}
	if res.RoomNumber != "301" || len(res.Receivers) != 1 {
		t.Fatalf("unexpected PairResult: %+v", res)
	}
	if len(fw.ActiveRules()) != 1 {
		t.Fatalf("expected 1 firewall rule, got %d", len(fw.ActiveRules()))
	}

	// Same IP re-pairs to Room 302 -> old rule to 301 must be revoked!
	if _, err := store.PairViaPortal("302", "Rahma", "10.10.30.5", ""); err != nil {
		t.Fatalf("re-pair to 302 failed: %v", err)
	}
	rules := fw.ActiveRules()
	if len(rules) != 1 || rules[0].ReceiverIP != "10.20.3.102" {
		t.Fatalf("expected old rule revoked and only 302 rule active, got %+v", rules)
	}

	// Segmented ads: Room 302 is corporate, should see corporate lounge ad, not leisure-only spa ad
	st302, err := store.TVState("tv-302-main")
	if err != nil {
		t.Fatalf("TVState 302 failed: %v", err)
	}
	for _, ad := range st302.Ads {
		if ad.ID == "ad-spa-20" {
			t.Fatalf("corporate room 302 should not receive leisure/vip-only ad")
		}
	}

	// Send room-targeted message to 302, then check out 302
	store.AddMessage(models.GuestMessage{
		RoomNumber:    "302",
		Title:         "Spa Ready",
		Body:          "Your spa appointment is at 5 PM.",
		Priority:      models.PriorityUrgent,
		ShowAsOverlay: true,
	})
	oldEpoch := st302.ResetEpoch

	cleared, err := store.CheckOut("302")
	if err != nil || cleared != 1 {
		t.Fatalf("expected 1 session cleared on checkout, got cleared=%d err=%v", cleared, err)
	}
	if len(fw.ActiveRules()) != 0 {
		t.Fatalf("expected 0 firewall rules after checkout, got %+v", fw.ActiveRules())
	}

	stAfter, err := store.TVState("tv-302-main")
	if err != nil {
		t.Fatalf("TVState after checkout failed: %v", err)
	}
	if stAfter.ResetEpoch != oldEpoch+1 {
		t.Fatalf("expected ResetEpoch increment from %d to %d", oldEpoch, stAfter.ResetEpoch)
	}
	if stAfter.Occupied || stAfter.GuestName != "" {
		t.Fatalf("expected vacant room after checkout, got %+v", stAfter)
	}
	// Room-specific message must be purged, while broadcast welcome remains
	for _, m := range stAfter.Messages {
		if m.Title == "Spa Ready" {
			t.Fatalf("room-specific message leaked after checkout")
		}
	}
}

func TestMaxDevicesEvictionAndJanitorExpiry(t *testing.T) {
	fw := firewall.NewMemoryDriver()
	store := NewStore(fw, nil)
	now := time.Date(2026, 11, 26, 14, 0, 0, 0, time.UTC)
	store.SetNowFunc(func() time.Time { return now })
	store.SeedDemoProperty()

	// Default MaxDevicesRoom = 5; bind 6 devices -> oldest gets evicted
	ips := []string{"10.10.1.1", "10.10.1.2", "10.10.1.3", "10.10.1.4", "10.10.1.5", "10.10.1.6"}
	for i, ip := range ips {
		now = now.Add(time.Duration(i+1) * time.Second)
		if _, err := store.PairViaPortal("301", "Budi", ip, ""); err != nil {
			t.Fatalf("pair %s failed: %v", ip, err)
		}
	}
	if len(fw.ActiveRules()) != 5 {
		t.Fatalf("expected 5 active rules after eviction, got %d", len(fw.ActiveRules()))
	}
	if room, _ := store.ReceiversForGuestIP("10.10.1.1"); room != nil {
		t.Fatalf("expected oldest guest IP 10.10.1.1 to be evicted")
	}

	// Advance past checkout tomorrow 12:00 -> janitor clears all 5
	expired := store.ExpireStale(now.Add(48 * time.Hour))
	if expired != 5 || len(fw.ActiveRules()) != 0 {
		t.Fatalf("expected 5 expired sessions and 0 rules, got expired=%d rules=%d", expired, len(fw.ActiveRules()))
	}
}

func TestCheckOut_TerminatesOnlyTargetRoomFlows(t *testing.T) {
	var commands []string
	fw := firewall.NewNftablesDriver(func(name string, args ...string) ([]byte, error) {
		commands = append(commands, name+" "+strings.Join(args, " "))
		return nil, nil
	})
	store := NewStore(fw, nil)
	store.SeedDemoProperty()

	// Bind one guest device in room 301 and one in room 302
	if _, err := store.PairViaPortal("301", "Budi", "10.10.30.11", "aa:bb:cc:dd:ee:01"); err != nil {
		t.Fatalf("PairViaPortal 301 failed: %v", err)
	}
	if _, err := store.PairViaPortal("302", "Siti", "10.10.30.22", "aa:bb:cc:dd:ee:02"); err != nil {
		t.Fatalf("PairViaPortal 302 failed: %v", err)
	}

	commands = nil // reset to observe only CheckOut("301") commands
	cleared, err := store.CheckOut("301")
	if err != nil || cleared != 1 {
		t.Fatalf("CheckOut 301 failed: cleared=%d err=%v", cleared, err)
	}

	var saw301Flush, saw302Flush bool
	for _, cmd := range commands {
		if strings.HasPrefix(cmd, "conntrack -D -s 10.10.30.11 ") {
			saw301Flush = true
		}
		if strings.Contains(cmd, "10.10.30.22") {
			saw302Flush = true
		}
	}
	if !saw301Flush {
		t.Fatalf("expected conntrack flush for room 301 guest 10.10.30.11, got commands: %v", commands)
	}
	if saw302Flush {
		t.Fatalf("room 301 checkout must not touch room 302 flow, got commands: %v", commands)
	}
}

func TestRoamingUpdate_ReplacesGuestIPAndFirewallRule(t *testing.T) {
	fw := firewall.NewMemoryDriver()
	store := NewStore(fw, nil)
	store.SeedDemoProperty()

	if _, err := store.PairViaPortal("301", "Budi", "10.10.30.10", "aa:bb:cc:00:00:01"); err != nil {
		t.Fatalf("PairViaPortal failed: %v", err)
	}

	// Bind RadiusSessionID via Start event
	if err := store.HandleRadiusAccounting(models.RadiusAccountingEvent{
		StatusType:      "Start",
		RadiusSessionID: "rad-301-1",
		RoomNumber:      "301",
		GuestIP:         "10.10.30.10",
		GuestMAC:        "aa:bb:cc:00:00:01",
	}); err != nil {
		t.Fatalf("HandleRadiusAccounting Start failed: %v", err)
	}

	// Roaming: IP changes from 10.10.30.10 -> 10.10.30.55
	if err := store.HandleRadiusAccounting(models.RadiusAccountingEvent{
		StatusType:      "Interim-Update",
		RadiusSessionID: "rad-301-1",
		RoomNumber:      "301",
		GuestIP:         "10.10.30.55",
		GuestMAC:        "aa:bb:cc:00:00:01",
	}); err != nil {
		t.Fatalf("HandleRadiusAccounting Interim-Update failed: %v", err)
	}

	if room, _ := store.ReceiversForGuestIP("10.10.30.10"); room != nil {
		t.Fatalf("expected old IP 10.10.30.10 to be revoked")
	}
	if room, rcvs := store.ReceiversForGuestIP("10.10.30.55"); room == nil || room.RoomNumber != "301" || len(rcvs) == 0 {
		t.Fatalf("expected new IP 10.10.30.55 to be active for room 301")
	}
	for _, r := range fw.ActiveRules() {
		if r.GuestIP == "10.10.30.10" {
			t.Fatalf("old firewall rule for 10.10.30.10 was not removed")
		}
	}
}

func TestMACRandomization_PreservesSessionByRadiusSessionID(t *testing.T) {
	fw := firewall.NewMemoryDriver()
	store := NewStore(fw, nil)
	store.SeedDemoProperty()

	if _, err := store.PairViaPortal("301", "Budi", "10.10.30.10", "aa:bb:cc:00:00:01"); err != nil {
		t.Fatalf("PairViaPortal failed: %v", err)
	}
	if err := store.HandleRadiusAccounting(models.RadiusAccountingEvent{
		StatusType:      "Start",
		RadiusSessionID: "rad-mac-1",
		GuestIP:         "10.10.30.10",
		GuestMAC:        "aa:bb:cc:00:00:01",
	}); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// MAC randomization on Interim-Update with same RadiusSessionID
	if err := store.HandleRadiusAccounting(models.RadiusAccountingEvent{
		StatusType:      "Interim-Update",
		RadiusSessionID: "rad-mac-1",
		GuestIP:         "10.10.30.10",
		GuestMAC:        "de:ad:be:ef:00:02",
	}); err != nil {
		t.Fatalf("Interim-Update failed: %v", err)
	}

	if room, _ := store.ReceiversForGuestIP("10.10.30.10"); room == nil || room.RoomNumber != "301" {
		t.Fatalf("expected session to remain active after MAC randomization")
	}
	_, sessions, _ := store.ListRoomsAndSessions()
	if len(sessions) != 1 || sessions[0].GuestMAC != "de:ad:be:ef:00:02" {
		t.Fatalf("expected 1 session with updated MAC, got %+v", sessions)
	}
}

func TestRadiusStop_RevokesOnlyEndedSession(t *testing.T) {
	fw := firewall.NewMemoryDriver()
	store := NewStore(fw, nil)
	store.SeedDemoProperty()

	if _, err := store.PairViaPortal("301", "Budi", "10.10.30.10", "aa:bb:cc:00:00:01"); err != nil {
		t.Fatalf("PairViaPortal 1 failed: %v", err)
	}
	if _, err := store.PairViaPortal("301", "Budi", "10.10.30.11", "aa:bb:cc:00:00:02"); err != nil {
		t.Fatalf("PairViaPortal 2 failed: %v", err)
	}
	_ = store.HandleRadiusAccounting(models.RadiusAccountingEvent{
		StatusType:      "Start",
		RadiusSessionID: "rad-1",
		GuestIP:         "10.10.30.10",
	})
	_ = store.HandleRadiusAccounting(models.RadiusAccountingEvent{
		StatusType:      "Start",
		RadiusSessionID: "rad-2",
		GuestIP:         "10.10.30.11",
	})

	if err := store.HandleRadiusAccounting(models.RadiusAccountingEvent{
		StatusType:      "Stop",
		RadiusSessionID: "rad-1",
		GuestIP:         "10.10.30.10",
	}); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if room, _ := store.ReceiversForGuestIP("10.10.30.10"); room != nil {
		t.Fatalf("expected stopped session 10.10.30.10 to be revoked")
	}
	if room, _ := store.ReceiversForGuestIP("10.10.30.11"); room == nil || room.RoomNumber != "301" {
		t.Fatalf("expected other device 10.10.30.11 in room 301 to remain active")
	}
}
