package edge

import (
	"path/filepath"
	"testing"

	"github.com/inaply/gateway/internal/firewall"
	"github.com/inaply/gateway/internal/session"
)

func TestAtomicEdgePersistAndRestore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "edge-state.json")

	fs := NewFileStore(path)
	fw1 := firewall.NewMemoryDriver()
	store1 := session.NewStore(fw1, fs.Save)
	store1.SeedDemoProperty()

	if _, err := store1.PairViaPortal("301", "Budi", "10.10.30.77", ""); err != nil {
		t.Fatalf("PairViaPortal failed: %v", err)
	}

	// Simulate gateway reboot: new Store + new MemoryDriver reading from edge file
	snap, err := fs.Load()
	if err != nil {
		t.Fatalf("fs.Load failed: %v", err)
	}
	fw2 := firewall.NewMemoryDriver()
	store2 := session.NewStore(fw2, fs.Save)
	if err := store2.LoadSnapshot(*snap); err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}

	room, rcvs := store2.ReceiversForGuestIP("10.10.30.77")
	if room == nil || room.RoomNumber != "301" || len(rcvs) != 1 {
		t.Fatalf("expected restored pairing for 10.10.30.77 -> 301, got room=%+v rcvs=%+v", room, rcvs)
	}
	rules := fw2.ActiveRules()
	if len(rules) != 1 || rules[0].GuestIP != "10.10.30.77" || rules[0].ReceiverIP != "10.20.3.101" {
		t.Fatalf("expected firewall rule restored after reboot, got %+v", rules)
	}
}
