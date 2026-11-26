package ssdp

import (
	"strings"
	"testing"

	"github.com/inaply/gateway/internal/session"
)

func TestSSDPScopedDIALResponse(t *testing.T) {
	store := session.NewStore(nil, nil)
	store.SeedDemoProperty()
	resp := NewResponder(store)

	search := "M-SEARCH * HTTP/1.1\r\nHOST: 239.255.255.250:1900\r\nMAN: \"ssdp:discover\"\r\nMX: 3\r\nST: urn:dial-multiscreen-org:service:dial:1\r\n\r\n"

	// Unpaired IP gets no response
	if _, ok := resp.HandleSearch("10.10.30.88", search); ok {
		t.Fatal("expected unpaired IP to receive no SSDP DIAL reply")
	}

	// Pair to Room 302 -> gets only Room 302 LOCATION
	if _, err := store.PairViaPortal("302", "Rahma", "10.10.30.88", ""); err != nil {
		t.Fatalf("PairViaPortal failed: %v", err)
	}
	replies, ok := resp.HandleSearch("10.10.30.88", search)
	if !ok || len(replies) != 1 {
		t.Fatalf("expected 1 SSDP reply, got ok=%v len=%d", ok, len(replies))
	}
	if !strings.Contains(replies[0], "http://10.20.3.102:8008/ssdp/device-desc.xml") {
		t.Fatalf("unexpected SSDP LOCATION: %s", replies[0])
	}
}
