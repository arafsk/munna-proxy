package discovery

import (
	"strings"
	"testing"
)

func TestInviteRoundTrip(t *testing.T) {
	original := Invite{
		Version:     1,
		HostID:      "abc123",
		Host:        "192.168.1.10:1080",
		Relay:       "relay.example.com:443",
		Fingerprint: "SHA256:abcdef",
		Token:       "single-use",
		Name:        "studio mac",
	}
	raw, err := original.String()
	if err != nil {
		t.Fatalf("String: %v", err)
	}
	if !strings.HasPrefix(raw, "munna://abc123@192.168.1.10:1080?") {
		t.Fatalf("unexpected invite: %s", raw)
	}

	parsed, err := ParseInvite(raw)
	if err != nil {
		t.Fatalf("ParseInvite: %v", err)
	}
	if parsed != original {
		t.Fatalf("parsed = %+v, want %+v", parsed, original)
	}
}

func TestInviteWithoutHost(t *testing.T) {
	original := Invite{HostID: "abc123", Relay: "relay.example.com:443"}
	raw, err := original.String()
	if err != nil {
		t.Fatalf("String: %v", err)
	}
	parsed, err := ParseInvite(raw)
	if err != nil {
		t.Fatalf("ParseInvite: %v", err)
	}
	if parsed.HostID != "abc123" || parsed.Host != "" || parsed.Relay != original.Relay {
		t.Fatalf("parsed = %+v", parsed)
	}
}

func TestInviteRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		value Invite
		raw   string
	}{
		{name: "missing host id", value: Invite{Host: "127.0.0.1:1080"}},
		{name: "bad host", value: Invite{HostID: "abc", Host: "127.0.0.1"}},
		{name: "bad scheme", raw: "https://example.com"},
		{name: "bad version", raw: "munna://abc?v=9"},
		{name: "missing host id uri", raw: "munna://?v=1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw := test.raw
			if raw == "" {
				var err error
				raw, err = test.value.String()
				if err == nil {
					t.Fatalf("String(%+v) = %q, want error", test.value, raw)
				}
				return
			}
			if _, err := ParseInvite(raw); err == nil {
				t.Fatalf("ParseInvite(%q) succeeded, want error", raw)
			}
		})
	}
}

func TestNewHostID(t *testing.T) {
	first, err := NewHostID()
	if err != nil {
		t.Fatalf("NewHostID: %v", err)
	}
	second, err := NewHostID()
	if err != nil {
		t.Fatalf("NewHostID: %v", err)
	}
	if len(first) != 26 {
		t.Fatalf("host ID length = %d, want 26", len(first))
	}
	if first == second {
		t.Fatal("host IDs must be unique")
	}
	if strings.ToLower(first) != first {
		t.Fatalf("host ID %q is not lowercase", first)
	}
}

func TestNewToken(t *testing.T) {
	first, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	second, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if len(first) != 22 {
		t.Fatalf("token length = %d, want 22", len(first))
	}
	if first == second {
		t.Fatal("tokens must be unique")
	}
}
