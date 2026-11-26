package firewall

import (
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func TestMemoryDriver(t *testing.T) {
	d := NewMemoryDriver()
	if err := d.AllowPair("invalid", "10.20.1.1"); err == nil {
		t.Fatal("expected error for invalid guest IP")
	}
	if err := d.AllowPair("10.10.1.2", "10.20.1.101"); err != nil {
		t.Fatalf("AllowPair failed: %v", err)
	}
	if err := d.AllowPair("10.10.1.1", "10.20.1.100"); err != nil {
		t.Fatalf("AllowPair failed: %v", err)
	}

	got := d.ActiveRules()
	if len(got) != 2 || got[0].GuestIP != "10.10.1.1" {
		t.Fatalf("unexpected active rules: %+v", got)
	}

	_ = d.RevokePair("10.10.1.2", "10.20.1.101")
	if len(d.ActiveRules()) != 1 {
		t.Fatalf("expected 1 rule remaining, got %d", len(d.ActiveRules()))
	}
}

func TestNftablesDriverCommands(t *testing.T) {
	var commands []string
	drv := NewNftablesDriver(func(name string, args ...string) ([]byte, error) {
		commands = append(commands, name+" "+strings.Join(args, " "))
		return nil, nil
	})

	if err := drv.AllowPair("10.10.30.10", "10.20.3.101"); err != nil {
		t.Fatalf("AllowPair failed: %v", err)
	}
	if err := drv.RevokePair("10.10.30.10", "10.20.3.101"); err != nil {
		t.Fatalf("RevokePair failed: %v", err)
	}

	want := []string{
		"nft add element inet inaply authorized_cast_pairs { 10.10.30.10 . 10.20.3.101 }",
		"nft delete element inet inaply authorized_cast_pairs { 10.10.30.10 . 10.20.3.101 }",
		"conntrack -D -s 10.10.30.10 -d 10.20.3.101",
	}
	if !reflect.DeepEqual(commands, want) {
		t.Fatalf("commands mismatch\ngot:  %v\nwant: %v", commands, want)
	}
}

func TestNftablesDriver_RevokePair_CallsConntrackFlush(t *testing.T) {
	var commands []string
	drv := NewNftablesDriver(func(name string, args ...string) ([]byte, error) {
		commands = append(commands, name+" "+strings.Join(args, " "))
		return nil, nil
	})

	if err := drv.RevokePair("10.10.30.10", "10.20.3.101"); err != nil {
		t.Fatalf("RevokePair failed: %v", err)
	}

	want := []string{
		"nft delete element inet inaply authorized_cast_pairs { 10.10.30.10 . 10.20.3.101 }",
		"conntrack -D -s 10.10.30.10 -d 10.20.3.101",
	}
	if !reflect.DeepEqual(commands, want) {
		t.Fatalf("commands mismatch\ngot:  %v\nwant: %v", commands, want)
	}
}

func TestNftablesDriver_RevokePair_IgnoresMissingConntrackFlow(t *testing.T) {
	t.Run("zero flow entries deleted", func(t *testing.T) {
		drv := NewNftablesDriver(func(name string, args ...string) ([]byte, error) {
			if name == "conntrack" {
				return []byte("conntrack v1.4.6 (conntrack-tools): 0 flow entries have been deleted.\n"), errors.New("exit status 1")
			}
			return nil, nil
		})
		if err := drv.RevokePair("10.10.30.10", "10.20.3.101"); err != nil {
			t.Fatalf("expected non-fatal error when 0 flow entries deleted, got: %v", err)
		}
	})

	t.Run("conntrack binary missing", func(t *testing.T) {
		drv := NewNftablesDriver(func(name string, args ...string) ([]byte, error) {
			if name == "conntrack" {
				return nil, exec.ErrNotFound
			}
			return nil, nil
		})
		if err := drv.RevokePair("10.10.30.10", "10.20.3.101"); err != nil {
			t.Fatalf("expected non-fatal error when conntrack binary is missing, got: %v", err)
		}
	})
}
