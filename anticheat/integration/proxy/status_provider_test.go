package proxy

import (
	"context"
	"testing"
)

func TestNetherNetStatusProviderUsesPingFunction(t *testing.T) {
	const address = "http://example.test:19132"
	provider := newNetherNetStatusProvider(address, func(_ context.Context, got string) ([]byte, error) {
		if got != address {
			t.Errorf("ping address = %q, want %q", got, address)
		}
		return []byte("MCPE;NetherNet server;0;1.0;4;20;0;world;Creative;1;19132;19132;0;"), nil
	})
	defer provider.Close()

	provider.updateStatus()
	got := provider.ServerStatus(0, 0)
	if got.ServerName != "NetherNet server" || got.PlayerCount != 4 || got.MaxPlayers != 20 || got.GameType != 1 {
		t.Fatalf("status = %+v, want NetherNet response", got)
	}
}

func TestNewNetherNetStatusProviderNormalizesAndValidatesAddress(t *testing.T) {
	provider, err := NewNetherNetStatusProvider("example.test:19132")
	if err != nil {
		t.Fatal(err)
	}
	if provider.address != "http://example.test:19132" {
		t.Fatalf("provider address = %q, want HTTP URL", provider.address)
	}
	if err := provider.Close(); err != nil {
		t.Fatal(err)
	}

	for _, address := range []string{
		"ftp://example.test:19132",
		"http://example.test:19132/path",
		"http://example.test:19132?query",
		"http://example.test:65536",
	} {
		if _, err := NewNetherNetStatusProvider(address); err == nil {
			t.Errorf("NewNetherNetStatusProvider(%q) succeeded, want invalid address error", address)
		}
	}
}
