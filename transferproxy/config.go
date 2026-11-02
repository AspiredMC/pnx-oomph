package proxy

import (
	"context"
	"log/slog"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Config configures a standalone Bedrock proxy.
type Config struct {
	// LocalAddress is the local HTTP signaling address for NetherNet, such as ":19132".
	LocalAddress string
	// RemoteAddress is the NetherNet signaling URL, such as "https://example.com:19132".
	RemoteAddress string
	Log           *slog.Logger
	Listen        minecraft.ListenConfig
	DialTimeout   time.Duration
	NewHandler    HandlerFactory

	// Dial may be set by advanced users to customise backend connections.
	Dial DialFunc
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
			IdentityData:         identity,
			ClientData:           client,
			KeepXBLIdentityData:  true,
			DisablePortFollowing: true,
		}.DialContext(ctx, "nethernet", nethernetAddress(address))
	}
}
