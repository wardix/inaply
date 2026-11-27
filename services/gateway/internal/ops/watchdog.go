package ops

import (
	"sync"

	"github.com/inaply/gateway/internal/session"
)

// ProbeFunc checks whether a receiver IP is reachable on the LAN.
type ProbeFunc func(ip string) bool

// Watchdog periodically probes all room receivers and updates their status.
type Watchdog struct {
	mu        sync.Mutex
	store     *session.Store
	threshold int
	probe     ProbeFunc
	failures  map[string]int
}

func NewWatchdog(store *session.Store, threshold int, probe ProbeFunc) *Watchdog {
	if threshold <= 0 {
		threshold = 3
	}
	return &Watchdog{
		store:     store,
		threshold: threshold,
		probe:     probe,
		failures:  make(map[string]int),
	}
}

func (w *Watchdog) ProbeOnce() {
	_, _, receivers := w.store.ListRoomsAndSessions()
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, rcv := range receivers {
		if w.probe(rcv.IP) {
			w.failures[rcv.DeviceID] = 0
			if rcv.Status == "offline" || rcv.Status == "" {
				_ = w.store.Heartbeat(rcv.DeviceID, "idle", rcv.AppVersion, rcv.HDMICEC)
			}
			continue
		}
		w.failures[rcv.DeviceID]++
		if w.failures[rcv.DeviceID] >= w.threshold && rcv.Status != "offline" {
			_ = w.store.Heartbeat(rcv.DeviceID, "offline", rcv.AppVersion, rcv.HDMICEC)
		}
	}
}
