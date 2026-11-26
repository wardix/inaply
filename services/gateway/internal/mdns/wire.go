package mdns

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
)

const (
	TypeA    uint16 = 1
	TypePTR  uint16 = 12
	TypeTXT  uint16 = 16
	TypeAAAA uint16 = 28
	TypeSRV  uint16 = 33
	TypeANY  uint16 = 255

	ClassIN      uint16 = 1
	ClassINFlush uint16 = 0x8001
)

type Question struct {
	Name   string
	QType  uint16
	QClass uint16
	Unicast bool
}

type Record struct {
	Name       string
	Type       uint16
	Class      uint16
	TTL        uint32
	PTRTarget  string
	SRVTarget  string
	SRVPort    uint16
	SRVPrio    uint16
	SRVWeight  uint16
	TXTStrings []string
	IP         net.IP
	RawData    []byte
}

type Message struct {
	ID         uint16
	IsResponse bool
	Authoritative bool
	Questions  []Question
	Answers    []Record
	Additional []Record
}

func normalizeName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return "."
	}
	if !strings.HasSuffix(name, ".") {
		name += "."
	}
	return name
}

func DecodeMessage(pkt []byte) (*Message, error) {
	if len(pkt) < 12 {
		return nil, errors.New("mdns packet shorter than 12-byte header")
	}

	flags := binary.BigEndian.Uint16(pkt[2:4])
	qdCount := int(binary.BigEndian.Uint16(pkt[4:6]))
	anCount := int(binary.BigEndian.Uint16(pkt[6:8]))
	nsCount := int(binary.BigEndian.Uint16(pkt[8:10]))
	arCount := int(binary.BigEndian.Uint16(pkt[10:12]))

	msg := &Message{
		ID:            binary.BigEndian.Uint16(pkt[0:2]),
		IsResponse:    (flags & 0x8000) != 0,
		Authoritative: (flags & 0x0400) != 0,
	}

	offset := 12
	for i := 0; i < qdCount; i++ {
		name, next, err := decodeName(pkt, offset)
		if err != nil {
			return nil, fmt.Errorf("question %d name: %w", i, err)
		}
		if next+4 > len(pkt) {
			return nil, errors.New("truncated question")
		}
		qtype := binary.BigEndian.Uint16(pkt[next : next+2])
		rawClass := binary.BigEndian.Uint16(pkt[next+2 : next+4])
		msg.Questions = append(msg.Questions, Question{
			Name:    normalizeName(name),
			QType:   qtype,
			QClass:  rawClass & 0x7FFF,
			Unicast: (rawClass & 0x8000) != 0,
		})
		offset = next + 4
	}

	var err error
	for i := 0; i < anCount; i++ {
		var rec Record
		rec, offset, err = decodeRecord(pkt, offset)
		if err != nil {
			return nil, fmt.Errorf("answer %d: %w", i, err)
		}
		msg.Answers = append(msg.Answers, rec)
	}
	for i := 0; i < nsCount; i++ {
		_, offset, err = decodeRecord(pkt, offset)
		if err != nil {
			return nil, fmt.Errorf("authority %d: %w", i, err)
		}
	}
	for i := 0; i < arCount; i++ {
		var rec Record
		rec, offset, err = decodeRecord(pkt, offset)
		if err != nil {
			return nil, fmt.Errorf("additional %d: %w", i, err)
		}
		msg.Additional = append(msg.Additional, rec)
	}

	return msg, nil
}

func decodeRecord(pkt []byte, offset int) (Record, int, error) {
	name, next, err := decodeName(pkt, offset)
	if err != nil {
		return Record{}, 0, err
	}
	if next+10 > len(pkt) {
		return Record{}, 0, errors.New("truncated RR header")
	}
	rtype := binary.BigEndian.Uint16(pkt[next : next+2])
	rclass := binary.BigEndian.Uint16(pkt[next+2 : next+4])
	ttl := binary.BigEndian.Uint32(pkt[next+4 : next+8])
	rdLen := int(binary.BigEndian.Uint16(pkt[next+8 : next+10]))
	rdataStart := next + 10
	rdataEnd := rdataStart + rdLen
	if rdataEnd > len(pkt) {
		return Record{}, 0, errors.New("truncated RR rdata")
	}

	rec := Record{
		Name:    normalizeName(name),
		Type:    rtype,
		Class:   rclass,
		TTL:     ttl,
		RawData: append([]byte(nil), pkt[rdataStart:rdataEnd]...),
	}

	switch rtype {
	case TypePTR:
		target, _, err := decodeName(pkt, rdataStart)
		if err != nil {
			return Record{}, 0, err
		}
		rec.PTRTarget = normalizeName(target)
	case TypeSRV:
		if rdLen < 6 {
			return Record{}, 0, errors.New("invalid SRV rdata length")
		}
		rec.SRVPrio = binary.BigEndian.Uint16(pkt[rdataStart : rdataStart+2])
		rec.SRVWeight = binary.BigEndian.Uint16(pkt[rdataStart+2 : rdataStart+4])
		rec.SRVPort = binary.BigEndian.Uint16(pkt[rdataStart+4 : rdataStart+6])
		target, _, err := decodeName(pkt, rdataStart+6)
		if err != nil {
			return Record{}, 0, err
		}
		rec.SRVTarget = normalizeName(target)
	case TypeA:
		if rdLen == 4 {
			rec.IP = net.IPv4(pkt[rdataStart], pkt[rdataStart+1], pkt[rdataStart+2], pkt[rdataStart+3])
		}
	case TypeTXT:
		p := rdataStart
		for p < rdataEnd {
			l := int(pkt[p])
			p++
			if p+l > rdataEnd {
				return Record{}, 0, errors.New("invalid TXT string length")
			}
			if l > 0 {
				rec.TXTStrings = append(rec.TXTStrings, string(pkt[p:p+l]))
			}
			p += l
		}
	}

	return rec, rdataEnd, nil
}

func decodeName(pkt []byte, start int) (string, int, error) {
	if start >= len(pkt) {
		return "", 0, errors.New("offset out of bounds")
	}
	var labels []string
	pos := start
	nextOffset := 0
	jumped := false
	jumps := 0

	for {
		if pos >= len(pkt) {
			return "", 0, errors.New("unexpected end of name")
		}
		b := pkt[pos]
		if b == 0 {
			if !jumped {
				nextOffset = pos + 1
			}
			break
		}
		if (b & 0xC0) == 0xC0 {
			if pos+1 >= len(pkt) {
				return "", 0, errors.New("truncated compression pointer")
			}
			ptr := int(b&0x3F)<<8 | int(pkt[pos+1])
			if ptr >= len(pkt) {
				return "", 0, errors.New("invalid compression pointer")
			}
			if !jumped {
				nextOffset = pos + 2
				jumped = true
			}
			jumps++
			if jumps > 16 {
				return "", 0, errors.New("compression pointer loop")
			}
			pos = ptr
			continue
		}
		if (b & 0xC0) != 0 {
			return "", 0, errors.New("unsupported DNS label type")
		}
		l := int(b)
		pos++
		if pos+l > len(pkt) {
			return "", 0, errors.New("truncated DNS label")
		}
		labels = append(labels, string(pkt[pos:pos+l]))
		pos += l
	}

	if len(labels) == 0 {
		return ".", nextOffset, nil
	}
	return strings.Join(labels, ".") + ".", nextOffset, nil
}

func EncodeMessage(msg Message) ([]byte, error) {
	buf := make([]byte, 12)
	binary.BigEndian.PutUint16(buf[0:2], msg.ID)

	var flags uint16
	if msg.IsResponse {
		flags |= 0x8000
	}
	if msg.Authoritative {
		flags |= 0x0400
	}
	binary.BigEndian.PutUint16(buf[2:4], flags)
	binary.BigEndian.PutUint16(buf[4:6], uint16(len(msg.Questions)))
	binary.BigEndian.PutUint16(buf[6:8], uint16(len(msg.Answers)))
	binary.BigEndian.PutUint16(buf[8:10], 0)
	binary.BigEndian.PutUint16(buf[10:12], uint16(len(msg.Additional)))

	for _, q := range msg.Questions {
		nameBytes, err := encodeName(q.Name)
		if err != nil {
			return nil, err
		}
		buf = append(buf, nameBytes...)
		qclass := q.QClass
		if qclass == 0 {
			qclass = ClassIN
		}
		if q.Unicast {
			qclass |= 0x8000
		}
		var tmp [4]byte
		binary.BigEndian.PutUint16(tmp[0:2], q.QType)
		binary.BigEndian.PutUint16(tmp[2:4], qclass)
		buf = append(buf, tmp[:]...)
	}

	appendRecords := func(recs []Record) error {
		for _, r := range recs {
			encoded, err := encodeRecord(r)
			if err != nil {
				return err
			}
			buf = append(buf, encoded...)
		}
		return nil
	}

	if err := appendRecords(msg.Answers); err != nil {
		return nil, err
	}
	if err := appendRecords(msg.Additional); err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeRecord(r Record) ([]byte, error) {
	nameBytes, err := encodeName(r.Name)
	if err != nil {
		return nil, err
	}
	rdata, err := encodeRData(r)
	if err != nil {
		return nil, err
	}
	class := r.Class
	if class == 0 {
		class = ClassIN
	}
	var hdr [10]byte
	binary.BigEndian.PutUint16(hdr[0:2], r.Type)
	binary.BigEndian.PutUint16(hdr[2:4], class)
	binary.BigEndian.PutUint32(hdr[4:8], r.TTL)
	binary.BigEndian.PutUint16(hdr[8:10], uint16(len(rdata)))

	out := make([]byte, 0, len(nameBytes)+10+len(rdata))
	out = append(out, nameBytes...)
	out = append(out, hdr[:]...)
	out = append(out, rdata...)
	return out, nil
}

func encodeRData(r Record) ([]byte, error) {
	switch r.Type {
	case TypePTR:
		return encodeName(r.PTRTarget)
	case TypeSRV:
		targetBytes, err := encodeName(r.SRVTarget)
		if err != nil {
			return nil, err
		}
		out := make([]byte, 6+len(targetBytes))
		binary.BigEndian.PutUint16(out[0:2], r.SRVPrio)
		binary.BigEndian.PutUint16(out[2:4], r.SRVWeight)
		binary.BigEndian.PutUint16(out[4:6], r.SRVPort)
		copy(out[6:], targetBytes)
		return out, nil
	case TypeTXT:
		if len(r.TXTStrings) == 0 {
			return []byte{0}, nil
		}
		var out []byte
		for _, s := range r.TXTStrings {
			if len(s) > 255 {
				return nil, fmt.Errorf("TXT string exceeds 255 bytes: %q", s)
			}
			out = append(out, byte(len(s)))
			out = append(out, s...)
		}
		return out, nil
	case TypeA:
		ip4 := r.IP.To4()
		if ip4 == nil {
			return nil, fmt.Errorf("invalid IPv4 for A record: %v", r.IP)
		}
		return []byte(ip4), nil
	default:
		return r.RawData, nil
	}
}

func encodeName(name string) ([]byte, error) {
	trimmed := strings.Trim(name, ".")
	if trimmed == "" {
		return []byte{0}, nil
	}
	parts := strings.Split(trimmed, ".")
	var out []byte
	for _, label := range parts {
		if len(label) == 0 || len(label) > 63 {
			return nil, fmt.Errorf("invalid DNS label %q in %q", label, name)
		}
		out = append(out, byte(len(label)))
		out = append(out, label...)
	}
	out = append(out, 0)
	return out, nil
}
