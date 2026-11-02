package proxy

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
)

// NetherNetStatusProvider polls a NetherNet backend for its server-list status.
type NetherNetStatusProvider struct {
	address string
	ping    func(context.Context, string) ([]byte, error)

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	mu     sync.RWMutex
	status minecraft.ServerStatus
}

// NewNetherNetStatusProvider creates a status provider for a NetherNet backend.
// Bare host:port addresses are treated as HTTPS signaling URLs.
func NewNetherNetStatusProvider(address string) (*NetherNetStatusProvider, error) {
	if address != "" && !strings.Contains(address, "://") {
		address = "https://" + address
	}
	if err := validateNetherNetStatusAddress(address); err != nil {
		return nil, err
	}
	return newNetherNetStatusProvider(address, minecraft.NetherNet{}.PingContext), nil
}

func newNetherNetStatusProvider(address string, ping func(context.Context, string) ([]byte, error)) *NetherNetStatusProvider {
	ctx, cancel := context.WithCancel(context.Background())
	provider := &NetherNetStatusProvider{
		address: address,
		ping:    ping,
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
	}
	go provider.update()
	return provider
}

func validateNetherNetStatusAddress(address string) error {
	u, err := url.Parse(address)
	if err != nil {
		return fmt.Errorf("invalid NetherNet status address %q: %w", address, err)
	}
	port, portErr := strconv.Atoi(u.Port())
	if (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.Port() == "" ||
		portErr != nil || port < 1 || port > 65535 || u.User != nil ||
		(u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("invalid NetherNet status address %q", address)
	}
	return nil
}

// ServerStatus returns the most recently fetched backend status.
func (p *NetherNetStatusProvider) ServerStatus(int, int) minecraft.ServerStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// Close stops the background status poller.
func (p *NetherNetStatusProvider) Close() error {
	p.cancel()
	<-p.done
	return nil
}

func (p *NetherNetStatusProvider) update() {
	defer close(p.done)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		p.updateStatus()
		select {
		case <-ticker.C:
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *NetherNetStatusProvider) updateStatus() {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()
	data, err := p.ping(ctx, p.address)
	if err != nil {
		return
	}
	p.mu.Lock()
	p.status = minecraft.ParsePongData(data)
	p.mu.Unlock()
}
