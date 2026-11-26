package ssdp

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/inaply/gateway/internal/mdns"
)

const DIALSearchTarget = "urn:dial-multiscreen-org:service:dial:1"

// Responder handles SSDP M-SEARCH discovery (UDP 1900) and replies solely
// with DIAL endpoints belonging to the querying guest IP's paired room.
type Responder struct {
	lookup mdns.RoomLookup
}

func NewResponder(lookup mdns.RoomLookup) *Responder {
	return &Responder{lookup: lookup}
}

func (r *Responder) HandleSearch(srcIP, payload string) ([]string, bool) {
	upper := strings.ToUpper(payload)
	if !strings.HasPrefix(upper, "M-SEARCH") {
		return nil, false
	}

	st := extractHeader(payload, "ST")
	if st == "" {
		return nil, false
	}
	stLower := strings.ToLower(st)
	if stLower != "ssdp:all" && stLower != strings.ToLower(DIALSearchTarget) {
		return nil, false
	}

	room, receivers := r.lookup.ReceiversForGuestIP(srcIP)
	if room == nil || len(receivers) == 0 {
		return nil, false
	}

	responses := make([]string, 0, len(receivers))
	for _, rcv := range receivers {
		port := rcv.CastPort
		if port == 0 {
			port = 8008
		}
		// DIAL XML typically served on 8008 for Google Cast devices.
		dialPort := 8008
		if port != 8009 {
			dialPort = port
		}
		resp := fmt.Sprintf(
			"HTTP/1.1 200 OK\r\n"+
				"CACHE-CONTROL: max-age=45\r\n"+
				"EXT:\r\n"+
				"LOCATION: http://%s:%d/ssdp/device-desc.xml\r\n"+
				"SERVER: Linux/5.15 UPnP/1.1 InaplyGateway/1.0\r\n"+
				"ST: %s\r\n"+
				"USN: uuid:%s::%s\r\n"+
				"X-ROOM-NUMBER: %s\r\n"+
				"X-DEVICE-LABEL: %s\r\n\r\n",
			rcv.IP, dialPort, DIALSearchTarget, rcv.DeviceID, DIALSearchTarget, room.RoomNumber, rcv.Label,
		)
		responses = append(responses, resp)
	}
	return responses, true
}

func (r *Responder) Serve(ctx context.Context, listenAddr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp4", listenAddr)
	if err != nil {
		return err
	}
	var conn *net.UDPConn
	if udpAddr.Port == 1900 && (udpAddr.IP == nil || udpAddr.IP.IsUnspecified()) {
		mcastAddr := &net.UDPAddr{IP: net.IPv4(239, 255, 255, 250), Port: 1900}
		conn, err = net.ListenMulticastUDP("udp4", nil, mcastAddr)
		if err != nil {
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

	buf := make([]byte, 2048)
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

		resps, ok := r.HandleSearch(peer.IP.String(), string(buf[:n]))
		if !ok {
			continue
		}
		for _, payload := range resps {
			if _, err := conn.WriteToUDP([]byte(payload), peer); err != nil && ctx.Err() == nil {
				log.Printf("[ssdp] write error to %s: %v", peer, err)
			}
		}
	}
}

func extractHeader(payload, header string) string {
	lines := strings.Split(payload, "\n")
	prefix := strings.ToLower(header) + ":"
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), prefix) {
			return strings.TrimSpace(trimmed[len(prefix):])
		}
	}
	return ""
}
