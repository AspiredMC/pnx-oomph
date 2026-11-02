package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	"github.com/oomph-ac/oomph/anticheat/integration/proxy"
	"github.com/oomph-ac/oomph/anticheat/utils"
	"github.com/sandertv/gophertunnel/minecraft"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: oomph-proxy <local-address> <remote-address>")
		fmt.Println("Example: oomph-proxy :19132 http://127.0.0.1:19133")
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	remoteAddress := os.Args[2]
	if !strings.Contains(remoteAddress, "://") {
		remoteAddress = "http://" + remoteAddress
	}
	status, err := proxy.NewNetherNetStatusProvider(remoteAddress)
	if err != nil {
		panic(err)
	}
	defer status.Close()
	utils.InitializeBlockNameMapping()
	p, err := proxy.Listen(ctx, proxy.Config{
		LocalAddress:  os.Args[1],
		RemoteAddress: remoteAddress,
		Log:           slog.Default(),
		Listen: minecraft.ListenConfig{
			StatusProvider:      status,
			FlushRate:           -1,
			AllowUnknownPackets: true,
			AllowInvalidPackets: true,
		},
	})
	if err != nil {
		panic(err)
	}
	defer p.Close()
	if err := p.Serve(ctx); err != nil && ctx.Err() == nil {
		panic(err)
	}
}
