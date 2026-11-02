package proxy

import (
	"context"
	"fmt"

	"github.com/df-mc/go-nethernet"
)

func netherNetCredentialsProvider(configured func(context.Context) (*nethernet.Credentials, error)) func(context.Context) (*nethernet.Credentials, error) {
	return func(ctx context.Context) (*nethernet.Credentials, error) {
		credentials := &nethernet.Credentials{
			ICEServers: []nethernet.ICEServer{
				{URLs: []string{"stun:stun.cloudflare.com:3478"}},
				{URLs: []string{"stun:stun.l.google.com:19302"}},
			},
		}
		if configured == nil {
			return credentials, nil
		}
		extra, err := configured(ctx)
		if err != nil {
			return nil, fmt.Errorf("proxy: load additional NetherNet ICE servers: %w", err)
		}
		if extra == nil {
			return credentials, nil
		}
		credentials.ExpirationInSeconds = extra.ExpirationInSeconds
		credentials.ICEServers = append(credentials.ICEServers, extra.ICEServers...)
		return credentials, nil
	}
}
