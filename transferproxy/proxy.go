// Package proxy provides a standalone Bedrock proxy with a NetherNet listener.
package proxy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/df-mc/go-nethernet"
	"github.com/df-mc/go-nethernet/endpoint"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Proxy accepts Bedrock clients and keeps them connected while their backend
// connection is replaced during packet.Transfer handoffs.
type Proxy struct {
	cfg       Config
	listener  *minecraft.Listener
	signaling *endpoint.Handler
	done      chan struct{}
	closeOne  sync.Once
	closeErr  error

	batchReading bool
}

// Listen starts a standalone Bedrock proxy.
func Listen(ctx context.Context, cfg Config) (*Proxy, error) {
	if cfg.LocalAddress == "" || cfg.RemoteAddress == "" {
		return nil, fmt.Errorf("proxy: local and remote addresses are required")
	}
	if cfg.Log == nil {
		cfg.Log = slog.Default()
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 10 * time.Second
	}
	if cfg.Dial == nil {
		cfg.Dial = defaultDial(cfg.DialTimeout)
	}
	if cfg.Listen.Compression == nil {
		cfg.Listen.Compression = packet.SnappyCompression
	}
	identityFile := cfg.NetherNetIdentityFile
	if identityFile == "" {
		identityFile = filepath.Join("Keys", "identity.pem")
	}
	identityKey, generated, err := ensureNetherNetIdentity(identityFile)
	if err != nil {
		return nil, err
	}
	if generated {
		cfg.Log.Info("generated a NetherNet identity; keep this file to preserve client trust", "file", identityFile)
	}
	credentials := netherNetCredentialsProvider(cfg.NetherNetCredentials)
	if cfg.NetherNetListenConfig.API != nil {
		credentials = cfg.NetherNetCredentials
	}
	server, err := (endpoint.HandlerConfig{
		Credentials:        credentials,
		NegotiationContext: cfg.NetherNetListenConfig.NegotiationContext,
	}).Serve(cfg.LocalAddress)
	if err != nil {
		return nil, fmt.Errorf("proxy: start signaling server: %w", err)
	}
	domain := cfg.NetherNetIdentityDomain
	if domain == "" {
		domain = "self"
	}
	netherNetNetwork := minecraft.NetherNet{
		Signaling:    httpSignaling{Handler: server},
		ListenConfig: cfg.NetherNetListenConfig,
	}
	if netherNetNetwork.ListenConfig.IssueServerIdentity == nil {
		netherNetNetwork.ListenConfig.IssueServerIdentity = func(context.Context) (*nethernet.Identity, error) {
			return nethernet.ServerIdentity(identityKey, domain)
		}
	}
	l, err := cfg.Listen.ListenNetwork(netherNetNetwork, server.NetworkID())
	if err != nil {
		return nil, errors.Join(fmt.Errorf("proxy: listen: %w", err), server.Close())
	}
	p := &Proxy{cfg: cfg, listener: l, signaling: server, done: make(chan struct{})}
	go func() {
		select {
		case <-ctx.Done():
			_ = p.Close()
		case <-p.done:
		}
	}()
	return p, nil
}

// Serve accepts clients until the proxy is closed.
func (p *Proxy) Serve(ctx context.Context) error {
	for {
		raw, err := p.listener.Accept()
		if err != nil {
			select {
			case <-p.done:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			default:
				return fmt.Errorf("proxy: accept: %w", err)
			}
		}
		conn := raw.(*minecraft.Conn)
		go func() {
			if err := p.serveClient(ctx, conn); err != nil {
				p.cfg.Log.Error("proxy session closed", "player", conn.IdentityData().DisplayName, "err", err)
			}
		}()
	}
}

func (p *Proxy) serveClient(ctx context.Context, conn *minecraft.Conn) error {
	clientData := conn.ClientData()
	clientData.ThirdPartyName = conn.IdentityData().DisplayName
	backend, err := p.cfg.Dial(ctx, p.cfg.RemoteAddress, conn.IdentityData(), clientData, conn.RemoteAddr().String())
	if err != nil {
		reason := "Unable to connect to the backend server."
		if backendReason, ok := backendDisconnectMessage(err); ok {
			reason = backendReason
		}
		_ = p.listener.Disconnect(conn, reason)
		return err
	}
	handler := Handler(NopHandler{})
	if p.cfg.NewHandler != nil {
		handler = p.cfg.NewHandler(HandlerContext{
			Client: conn, Listener: p.listener,
			Log: p.cfg.Log.With("player", conn.IdentityData().DisplayName),
		})
		if handler == nil {
			handler = NopHandler{}
		}
	}
	s := newSession(p, handler, conn, backend, conn.IdentityData(), clientData, conn.RemoteAddr().String())
	defer s.close()
	return s.run(ctx)
}

// Close stops accepting new clients.
func (p *Proxy) Close() error {
	p.closeOne.Do(func() {
		close(p.done)
		p.closeErr = errors.Join(p.listener.Close(), p.signaling.Close())
	})
	return p.closeErr
}

type httpSignaling struct {
	*endpoint.Handler
}

func (s httpSignaling) Signal(ctx context.Context, signal *nethernet.Signal) error {
	err := s.Handler.Signal(ctx, signal)
	if err != nil && signal != nil && signal.Type == nethernet.SignalTypeError && strings.HasPrefix(err.Error(), "unexpected connection ID:") {
		return nil
	}
	return err
}
