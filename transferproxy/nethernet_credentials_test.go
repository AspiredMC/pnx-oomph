package proxy

import (
	"context"
	"errors"
	"testing"

	"github.com/df-mc/go-nethernet"
)

func TestNetherNetCredentialsProviderIncludesDefaultSTUN(t *testing.T) {
	credentials, err := netherNetCredentialsProvider(nil)(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(credentials.ICEServers) != 2 {
		t.Fatalf("ICE server count = %d, want 2", len(credentials.ICEServers))
	}
	if got, want := credentials.ICEServers[0].URLs[0], "stun:stun.cloudflare.com:3478"; got != want {
		t.Fatalf("first STUN server = %q, want %q", got, want)
	}
	if got, want := credentials.ICEServers[1].URLs[0], "stun:stun.l.google.com:19302"; got != want {
		t.Fatalf("second STUN server = %q, want %q", got, want)
	}
}

func TestNetherNetCredentialsProviderAppendsConfiguredServers(t *testing.T) {
	provider := netherNetCredentialsProvider(func(context.Context) (*nethernet.Credentials, error) {
		return &nethernet.Credentials{
			ExpirationInSeconds: 120,
			ICEServers: []nethernet.ICEServer{{
				Username: "user",
				Password: "secret",
				URLs:     []string{"turn:relay.example:3478"},
			}},
		}, nil
	})
	credentials, err := provider(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if credentials.ExpirationInSeconds != 120 {
		t.Fatalf("credential expiration = %d, want 120", credentials.ExpirationInSeconds)
	}
	if len(credentials.ICEServers) != 3 {
		t.Fatalf("ICE server count = %d, want default STUN and configured TURN", len(credentials.ICEServers))
	}
}

func TestNetherNetCredentialsProviderReturnsConfiguredError(t *testing.T) {
	want := errors.New("credentials unavailable")
	provider := netherNetCredentialsProvider(func(context.Context) (*nethernet.Credentials, error) {
		return nil, want
	})
	if _, err := provider(context.Background()); !errors.Is(err, want) {
		t.Fatalf("provider error = %v, want %v", err, want)
	}
}
