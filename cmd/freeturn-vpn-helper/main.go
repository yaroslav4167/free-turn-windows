//go:build linux

// freeturn-vpn-helper is a headless, socket-activated network helper. It does
// not embed the GUI, load user plugins, or accept arbitrary executable paths.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/samosvalishe/free-turn-proxy/desktop/backend/wintunnel"
)

func main() {
	uid := flag.Uint("uid", 0, "UID authorized to use this service")
	flag.Parse()
	if *uid == 0 || uint64(*uid) > uint64(^uint32(0)) || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "a valid non-root --uid is required")
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	if err := wintunnel.RunActivatedHelper(ctx, uint32(*uid)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
