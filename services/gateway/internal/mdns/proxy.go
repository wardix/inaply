package mdns

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/inaply/gateway/internal/models"
)

const (
	ServicesMetaName = "_services._dns-sd._udp.local."
	GoogleCastSvc    = "_googlecast._tcp.local."
	AirPlaySvc       = "_airplay._tcp.local."
	RAOPSvc          = "_raop._tcp.local."
	DefaultTTLSec    = 45
)

// RoomLookup resolves which TV receivers a given guest source IP is authorized to discover.
type RoomLookup interface {
	ReceiversForGuestIP(guestIP string) (*models.Room, []models.ReceiverDevice)
}

// Proxy acts as a room-scoped mDNS responder/reflector. Unpaired guest IPs or guests
// querying from another room receive zero cross-room records.
type Proxy struct {
	lookup RoomLookup
	ttl    uint32
}

func NewProxy(lookup RoomLookup) *Proxy {
	return &Proxy{
		lookup: lookup,
		ttl:    DefaultTTLSec,
	}
}

// HandlePacket inspects an incoming mDNS query packet from srcIP and builds a
// room-isolated response packet when authorized receivers match the query.
func (p *Proxy) HandlePacket(srcIP string, raw []byte) ([]byte, bool, error) {
	msg, err := DecodeMessage(raw)
	if err != nil {
		return nil, false, err
	}
	// Ignore mDNS responses from clients to prevent reflection loops / rogue announcements.
	if msg.IsResponse || len(msg.Questions) == 0 {
		return nil, false, nil
	}

	room, receivers := p.lookup.ReceiversForGuestIP(srcIP)
	if room == nil || len(receivers) == 0 {
		return nil, false, nil
	}

	resp := Message{
		ID:            msg.ID,
		IsResponse:    true,
		Authoritative: true,
	}
	seenAns := make(map[string]struct{})
	seenAdd := make(map[string]struct{})

	addUnique := func(dst *[]Record, seen map[string]struct{}, rec Record) {
		key := fmt.Sprintf("%s|%d|%s|%s|%v", rec.Name, rec.Type, rec.PTRTarget, rec.SRVTarget, rec.IP)
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		*dst = append(*dst, rec)
	}

	for _, q := range msg.Questions {
		qname := normalizeName(q.Name)

		if qname == ServicesMetaName && (q.QType == TypePTR || q.QType == TypeANY) {
			addUnique(&resp.Answers, seenAns, Record{
				Name:      ServicesMetaName,
				Type:      TypePTR,
				Class:     ClassIN,
				TTL:       p.ttl,
				PTRTarget: GoogleCastSvc,
			})
			if hasAirPlay(receivers) {
				addUnique(&resp.Answers, seenAns, Record{
					Name:      ServicesMetaName,
					Type:      TypePTR,
					Class:     ClassIN,
					TTL:       p.ttl,
					PTRTarget: AirPlaySvc,
				})
			}
			continue
		}

		for _, rcv := range receivers {
			castInstance := castInstanceName(rcv)
			airplayInstance := airplayInstanceName(rcv)
			hostName := receiverHostName(rcv)
			ip := net.ParseIP(rcv.IP).To4()
			if ip == nil {
				continue
			}

			// Google Cast service PTR query
			if qname == GoogleCastSvc && (q.QType == TypePTR || q.QType == TypeANY) {
				addUnique(&resp.Answers, seenAns, Record{
					Name:      GoogleCastSvc,
					Type:      TypePTR,
					Class:     ClassIN,
					TTL:       p.ttl,
					PTRTarget: castInstance,
				})
				addUnique(&resp.Additional, seenAdd, castSRVRecord(rcv, castInstance, hostName, p.ttl))
				addUnique(&resp.Additional, seenAdd, castTXTRecord(rcv, castInstance, p.ttl))
				addUnique(&resp.Additional, seenAdd, aRecord(hostName, ip, p.ttl))
			}

			// Direct Cast instance SRV/TXT query
			if qname == castInstance {
				if q.QType == TypeSRV || q.QType == TypeANY {
					addUnique(&resp.Answers, seenAns, castSRVRecord(rcv, castInstance, hostName, p.ttl))
					addUnique(&resp.Additional, seenAdd, aRecord(hostName, ip, p.ttl))
				}
				if q.QType == TypeTXT || q.QType == TypeANY {
					addUnique(&resp.Answers, seenAns, castTXTRecord(rcv, castInstance, p.ttl))
				}
			}

			// AirPlay service PTR & instance queries
			if rcv.AirPlayEnabled {
				if qname == AirPlaySvc && (q.QType == TypePTR || q.QType == TypeANY) {
					addUnique(&resp.Answers, seenAns, Record{
						Name:      AirPlaySvc,
						Type:      TypePTR,
						Class:     ClassIN,
						TTL:       p.ttl,
						PTRTarget: airplayInstance,
					})
					addUnique(&resp.Additional, seenAdd, airplaySRVRecord(rcv, airplayInstance, hostName, p.ttl))
					addUnique(&resp.Additional, seenAdd, airplayTXTRecord(rcv, airplayInstance, p.ttl))
					addUnique(&resp.Additional, seenAdd, aRecord(hostName, ip, p.ttl))
				}
				if qname == airplayInstance {
					if q.QType == TypeSRV || q.QType == TypeANY {
						addUnique(&resp.Answers, seenAns, airplaySRVRecord(rcv, airplayInstance, hostName, p.ttl))
						addUnique(&resp.Additional, seenAdd, aRecord(hostName, ip, p.ttl))
					}
					if q.QType == TypeTXT || q.QType == TypeANY {
						addUnique(&resp.Answers, seenAns, airplayTXTRecord(rcv, airplayInstance, p.ttl))
					}
				}
			}

			// Direct .local hostname resolution
			if qname == hostName && (q.QType == TypeA || q.QType == TypeANY) {
				addUnique(&resp.Answers, seenAns, aRecord(hostName, ip, p.ttl))
			}
		}
	}

	if len(resp.Answers) == 0 {
		return nil, false, nil
	}
	encoded, err := EncodeMessage(resp)
	if err != nil {
		return nil, false, err
	}
	return encoded, true, nil
}

// Serve runs the mDNS UDP listener (multicast 224.0.0.251:5353 or unicast test socket).
func (p *Proxy) Serve(ctx context.Context, listenAddr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp4", listenAddr)
	if err != nil {
		return err
	}

	var conn *net.UDPConn
	if udpAddr.Port == 5353 && (udpAddr.IP == nil || udpAddr.IP.IsUnspecified()) {
		mcastAddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}
		conn, err = net.ListenMulticastUDP("udp4", nil, mcastAddr)
		if err != nil {
			// Fall back to standard UDP bind if multicast group join is unavailable.
			conn, err = net.ListenUDP("udp4", udpAddr)
		}
	} else {
		conn, err = net.ListenUDP("udp4", udpAddr)
	}
	if err != nil {
		return err
	}
	defer conn.Close()

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	buf := make([]byte, 4096)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, peer, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return err
		}

		resp, ok, err := p.HandlePacket(peer.IP.String(), buf[:n])
		if err != nil {
			log.Printf("[mdns] decode error from %s: %v", peer.IP, err)
			continue
		}
		if !ok {
			continue
		}
		// Reply directly via unicast to the querying guest IP so other rooms never sniff it.
		if _, err := conn.WriteToUDP(resp, peer); err != nil && ctx.Err() == nil {
			log.Printf("[mdns] write error to %s: %v", peer, err)
		}
	}
}

func hasAirPlay(receivers []models.ReceiverDevice) bool {
	for _, r := range receivers {
		if r.AirPlayEnabled {
			return true
		}
	}
	return false
}

func sanitizeLabel(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer(" ", "-", "_", "-", ".", "-")
	return replacer.Replace(s)
}

func receiverHostName(r models.ReceiverDevice) string {
	return normalizeName(sanitizeLabel(r.DeviceID) + ".local.")
}

func castInstanceName(r models.ReceiverDevice) string {
	return normalizeName(sanitizeLabel(r.DeviceID) + "." + GoogleCastSvc)
}

func airplayInstanceName(r models.ReceiverDevice) string {
	return normalizeName(sanitizeLabel(r.DeviceID) + "." + AirPlaySvc)
}

func castSRVRecord(r models.ReceiverDevice, instance, host string, ttl uint32) Record {
	port := uint16(r.CastPort)
	if port == 0 {
		port = 8009
	}
	return Record{
		Name:      instance,
		Type:      TypeSRV,
		Class:     ClassINFlush,
		TTL:       ttl,
		SRVPort:   port,
		SRVTarget: host,
	}
}

func castTXTRecord(r models.ReceiverDevice, instance string, ttl uint32) Record {
	model := r.ModelName
	if model == "" {
		model = "Google TV Streamer"
	}
	return Record{
		Name:  instance,
		Type:  TypeTXT,
		Class: ClassINFlush,
		TTL:   ttl,
		TXTStrings: []string{
			"id=" + r.DeviceID,
			"cd=" + r.DeviceID,
			"ve=05",
			"md=" + model,
			"fn=" + r.Label,
			"ca=4101",
			"st=0",
		},
	}
}

func airplaySRVRecord(r models.ReceiverDevice, instance, host string, ttl uint32) Record {
	port := uint16(r.AirPlayPort)
	if port == 0 {
		port = 7000
	}
	return Record{
		Name:      instance,
		Type:      TypeSRV,
		Class:     ClassINFlush,
		TTL:       ttl,
		SRVPort:   port,
		SRVTarget: host,
	}
}

func airplayTXTRecord(r models.ReceiverDevice, instance string, ttl uint32) Record {
	return Record{
		Name:  instance,
		Type:  TypeTXT,
		Class: ClassINFlush,
		TTL:   ttl,
		TXTStrings: []string{
			"deviceid=" + r.DeviceID,
			"features=0x5A7FFFF7,0x1E",
			"flags=0x4",
			"model=InaplyTV1,1",
			"srcvers=220.68",
		},
	}
}

func aRecord(host string, ip net.IP, ttl uint32) Record {
	return Record{
		Name:  host,
		Type:  TypeA,
		Class: ClassINFlush,
		TTL:   ttl,
		IP:    ip,
	}
}
