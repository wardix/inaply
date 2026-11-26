package firewall

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"sort"
	"strings"
	"sync"
)

// Rule represents an authorized L3 pair between a guest device IP and a room receiver IP.
type Rule struct {
	GuestIP    string `json:"guest_ip"`
	ReceiverIP string `json:"receiver_ip"`
}

// Driver abstracts L3/L4 isolation enforcement between Guest VLAN and Media/TV VLAN.
type Driver interface {
	AllowPair(guestIP, receiverIP string) error
	RevokePair(guestIP, receiverIP string) error
	ActiveRules() []Rule
}

// MemoryDriver enforces rule state in-memory for tests and non-root dry-run execution.
type MemoryDriver struct {
	mu    sync.RWMutex
	rules map[string]Rule
}

func NewMemoryDriver() *MemoryDriver {
	return &MemoryDriver{
		rules: make(map[string]Rule),
	}
}

func pairKey(guestIP, receiverIP string) string {
	return guestIP + "->" + receiverIP
}

func validateIPPair(guestIP, receiverIP string) error {
	if net.ParseIP(guestIP) == nil {
		return fmt.Errorf("invalid guest IP: %q", guestIP)
	}
	if net.ParseIP(receiverIP) == nil {
		return fmt.Errorf("invalid receiver IP: %q", receiverIP)
	}
	return nil
}

func (m *MemoryDriver) AllowPair(guestIP, receiverIP string) error {
	if err := validateIPPair(guestIP, receiverIP); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules[pairKey(guestIP, receiverIP)] = Rule{GuestIP: guestIP, ReceiverIP: receiverIP}
	return nil
}

func (m *MemoryDriver) RevokePair(guestIP, receiverIP string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rules, pairKey(guestIP, receiverIP))
	return nil
}

func (m *MemoryDriver) ActiveRules() []Rule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Rule, 0, len(m.rules))
	for _, r := range m.rules {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].GuestIP != out[j].GuestIP {
			return out[i].GuestIP < out[j].GuestIP
		}
		return out[i].ReceiverIP < out[j].ReceiverIP
	})
	return out
}

// CmdRunner executes OS commands (injectable for testing nftables commands without root).
type CmdRunner func(name string, args ...string) ([]byte, error)

// NftablesDriver manages elements in the `inet inaply authorized_cast_pairs` nft set.
type NftablesDriver struct {
	table  string
	set    string
	runner CmdRunner
	mem    *MemoryDriver
}

func NewNftablesDriver(runner CmdRunner) *NftablesDriver {
	if runner == nil {
		runner = func(name string, args ...string) ([]byte, error) {
			return exec.Command(name, args...).CombinedOutput()
		}
	}
	return &NftablesDriver{
		table:  "inaply",
		set:    "authorized_cast_pairs",
		runner: runner,
		mem:    NewMemoryDriver(),
	}
}

func (n *NftablesDriver) AllowPair(guestIP, receiverIP string) error {
	if err := validateIPPair(guestIP, receiverIP); err != nil {
		return err
	}
	expr := fmt.Sprintf("{ %s . %s }", guestIP, receiverIP)
	if out, err := n.runner("nft", "add", "element", "inet", n.table, n.set, expr); err != nil {
		return fmt.Errorf("nft add element failed: %w (%s)", err, string(out))
	}
	return n.mem.AllowPair(guestIP, receiverIP)
}

func (n *NftablesDriver) RevokePair(guestIP, receiverIP string) error {
	if err := validateIPPair(guestIP, receiverIP); err != nil {
		return err
	}
	expr := fmt.Sprintf("{ %s . %s }", guestIP, receiverIP)
	if out, err := n.runner("nft", "delete", "element", "inet", n.table, n.set, expr); err != nil {
		return fmt.Errorf("nft delete element failed: %w (%s)", err, string(out))
	}
	if err := n.flushConntrack(guestIP, receiverIP); err != nil {
		return err
	}
	return n.mem.RevokePair(guestIP, receiverIP)
}

func (n *NftablesDriver) flushConntrack(guestIP, receiverIP string) error {
	out, err := n.runner("conntrack", "-D", "-s", guestIP, "-d", receiverIP)
	if err == nil {
		return nil
	}
	// Non-fatal when conntrack-tools is not installed or no matching flow exists.
	if errors.Is(err, exec.ErrNotFound) || strings.Contains(string(out), "0 flow entries have been deleted") {
		return nil
	}
	return fmt.Errorf("conntrack flush failed: %w (%s)", err, string(out))
}

func (n *NftablesDriver) ActiveRules() []Rule {
	return n.mem.ActiveRules()
}
