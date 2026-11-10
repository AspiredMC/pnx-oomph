package proxy

import (
	"context"
	"log/slog"
	"time"

	"github.com/df-mc/go-nethernet"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Config configures a standalone Bedrock proxy.
type Config struct {
	// LocalAddress is the local HTTP signaling address to listen on, such as ":19132".
	LocalAddress string
	// RemoteAddress is the backend address passed to Dial or the default RakNet dialer.
	RemoteAddress string
	Log           *slog.Logger
	Listen        minecraft.ListenConfig
	DialTimeout   time.Duration
	NewHandler    HandlerFactory

	// Dial may be set by advanced users to customise backend connections.
	Dial DialFunc

	// NetherNetIdentityFile is the persistent PEM identity key file. If empty,
	// the proxy creates and reuses Keys/identity.pem.
	NetherNetIdentityFile string
	// NetherNetIdentityDomain is shown to players on first join; empty defaults to "self".
	NetherNetIdentityDomain string
	// NetherNetCredentials supplies additional ICE servers, such as TURN relays.
	// The proxy always includes its default public STUN servers as well.
	NetherNetCredentials func(context.Context) (*nethernet.Credentials, error)
	// NetherNetListenConfig customises ICE gathering and transport startup.
	// The proxy supplies the server identity when IssueServerIdentity is nil.
	NetherNetListenConfig nethernet.ListenConfig
}

// DialFunc establishes and logs into a backend, stopping before DoSpawn.
type DialFunc func(context.Context, string, login.IdentityData, login.ClientData, string) (Backend, error)

// Backend is the backend connection surface required by the native proxy.
type Backend interface {
	GameData() minecraft.GameData
	WritePacket(packet.Packet) error
	Close() error
	ReadPacket() (packet.Packet, error)
	DoSpawn() error
	Flush() error
}

func defaultDial(timeout time.Duration) DialFunc {
	return func(ctx context.Context, address string, identity login.IdentityData, client login.ClientData, _ string) (Backend, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return minecraft.Dialer{
			IdentityData:        identity,
			ClientData:          client,
			KeepXBLIdentityData: true,
		}.DialContext(ctx, "raknet", address)
	}
}
