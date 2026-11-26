package mdns

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/inaply/gateway/internal/firewall"
	"github.com/inaply/gateway/internal/session"
)

func buildQueryPacket(id uint16, name string, qtype uint16) []byte {
	pkt, _ := EncodeMessage(Message{
		ID: id,
		Questions: []Question{
			{Name: name, QType: qtype, QClass: ClassIN},
		},
	})
	return pkt
}

func TestWireEncodeDecodeWithCompression(t *testing.T) {
	// Raw DNS query with compression pointer in second question
	raw := []byte{
		0x12, 0x34, // ID
		0x00, 0x00, // Flags (query)
		0x00, 0x02, // QDCOUNT = 2
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Q1: _googlecast._tcp.local. PTR
		11, '_', 'g', 'o', 'o', 'g', 'l', 'e', 'c', 'a', 's', 't',
		4, '_', 't', 'c', 'p',
		5, 'l', 'o', 'c', 'a', 'l',
		0,
		0x00, 0x0c, 0x00, 0x01,
		// Q2: _airplay + pointer to offset 24 (._tcp.local.)
		8, '_', 'a', 'i', 'r', 'p', 'l', 'a', 'y',
		0xC0, 24,
		0x00, 0x0c, 0x00, 0x01,
	}

	msg, err := DecodeMessage(raw)
	if err != nil {
		t.Fatalf("DecodeMessage failed: %v", err)
	}
	if len(msg.Questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(msg.Questions))
	}
	if msg.Questions[0].Name != GoogleCastSvc {
		t.Fatalf("unexpected Q1 name: %s", msg.Questions[0].Name)
	}
	if msg.Questions[1].Name != AirPlaySvc {
		t.Fatalf("unexpected Q2 compressed name: %s", msg.Questions[1].Name)
	}
}

func TestRoomScopedMDNSIsolation(t *testing.T) {
	fw := firewall.NewMemoryDriver()
	store := session.NewStore(fw, nil)
	store.SeedDemoProperty()

	proxy := NewProxy(store)
	query := buildQueryPacket(42, GoogleCastSvc, TypePTR)

	// 1. Unpaired guest receives nothing.
	if _, ok, err := proxy.HandlePacket("10.10.30.99", query); err != nil || ok {
		t.Fatalf("expected unpaired guest to receive no mDNS response (ok=%v, err=%v)", ok, err)
	}

	// 2. Pair guest A to Room 301 via Captive Portal (surname match).
	if _, err := store.PairViaPortal("301", "Santoso", "10.10.30.10", ""); err != nil {
		t.Fatalf("PairViaPortal failed: %v", err)
	}

	rawRespA, ok, err := proxy.HandlePacket("10.10.30.10", query)
	if err != nil || !ok {
		t.Fatalf("expected Room 301 guest to get mDNS response, ok=%v err=%v", ok, err)
	}
	respA, err := DecodeMessage(rawRespA)
	if err != nil {
		t.Fatalf("DecodeMessage respA failed: %v", err)
	}
	if len(respA.Answers) != 1 || respA.Answers[0].PTRTarget != "tv-301-main._googlecast._tcp.local." {
		t.Fatalf("unexpected Room 301 answers: %+v", respA.Answers)
	}
	for _, add := range respA.Additional {
		if add.Type == TypeA && add.IP.String() != "10.20.3.101" {
			t.Fatalf("cross-room leak! saw IP %s", add.IP.String())
		}
	}

	// 3. Pair guest B to Suite 501 via fallback PIN -> sees BOTH Suite 501 TVs, none from 301/302.
	state501, err := store.TVState("tv-501-living")
	if err != nil {
		t.Fatalf("TVState failed: %v", err)
	}
	if _, err := store.PairViaPIN(state501.PairingPIN, "10.10.50.22", ""); err != nil {
		t.Fatalf("PairViaPIN failed: %v", err)
	}
	rawRespB, ok, err := proxy.HandlePacket("10.10.50.22", query)
	if err != nil || !ok {
		t.Fatalf("expected Suite 501 guest to get mDNS response, ok=%v err=%v", ok, err)
	}
	respB, err := DecodeMessage(rawRespB)
	if err != nil {
		t.Fatalf("DecodeMessage respB failed: %v", err)
	}
	if len(respB.Answers) != 2 {
		t.Fatalf("expected 2 PTR answers for Suite 501, got %d", len(respB.Answers))
	}

	// 4. Checkout Room 301 -> Guest A immediately loses visibility.
	if _, err := store.CheckOut("301"); err != nil {
		t.Fatalf("CheckOut failed: %v", err)
	}
	if _, ok, err := proxy.HandlePacket("10.10.30.10", query); err != nil || ok {
		t.Fatalf("expected checked-out guest to lose mDNS visibility (ok=%v, err=%v)", ok, err)
	}
}

func TestMDNSProxyUDPRoundTrip(t *testing.T) {
	store := session.NewStore(nil, nil)
	store.SeedDemoProperty()
	if _, err := store.PairViaPortal("301", "Budi", "127.0.0.1", ""); err != nil {
		t.Fatalf("PairViaPortal failed: %v", err)
	}

	proxy := NewProxy(store)

	// Probe an ephemeral free UDP port first
	probe, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("probe listen failed: %v", err)
	}
	srvAddr := probe.LocalAddr().String()
	_ = probe.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = proxy.Serve(ctx, srvAddr)
	}()
	time.Sleep(50 * time.Millisecond)

	c, err := net.Dial("udp4", srvAddr)
	if err != nil {
		t.Fatalf("dial udp failed: %v", err)
	}
	defer c.Close()

	_ = c.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := c.Write(buildQueryPacket(99, ServicesMetaName, TypePTR)); err != nil {
		t.Fatalf("write query failed: %v", err)
	}

	buf := make([]byte, 2048)
	n, err := c.Read(buf)
	if err != nil {
		t.Fatalf("read mDNS UDP response failed: %v", err)
	}
	msg, err := DecodeMessage(buf[:n])
	if err != nil {
		t.Fatalf("decode UDP response failed: %v", err)
	}
	if !msg.IsResponse || len(msg.Answers) < 2 {
		t.Fatalf("expected _googlecast and _airplay meta PTRs, got %+v", msg.Answers)
	}
}
