package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/inaply/gateway/internal/api"
	"github.com/inaply/gateway/internal/assets"
	"github.com/inaply/gateway/internal/edge"
	"github.com/inaply/gateway/internal/firewall"
	"github.com/inaply/gateway/internal/mdns"
	"github.com/inaply/gateway/internal/session"
	"github.com/inaply/gateway/internal/ssdp"
)

func main() {
	httpAddr := flag.String("addr", ":8080", "HTTP portal & API listen address")
	mdnsAddr := flag.String("mdns-addr", "127.0.0.1:5354", "mDNS proxy UDP listen address (use 0.0.0.0:5353 in production)")
	ssdpAddr := flag.String("ssdp-addr", "127.0.0.1:1901", "SSDP responder UDP listen address (use 0.0.0.0:1900 in production)")
	stateFile := flag.String("state-file", "./edge-state.json", "Path to atomic edge cache persistence file")
	fwMode := flag.String("firewall", "memory", "Firewall driver: memory | nftables")
	dryRun := flag.Bool("dry-run", true, "Run without modifying OS nftables (overrides -firewall to memory)")
	assetsDir := flag.String("assets-dir", "", "Directory for local LAN asset cache served at /assets/")
	trustedProxies := flag.String("trusted-proxies", "", "Comma-separated CIDRs or IPs of trusted reverse proxies for X-Forwarded-For")
	flag.Parse()

	var fw firewall.Driver
	if *dryRun || *fwMode != "nftables" {
		fw = firewall.NewMemoryDriver()
		log.Printf("[gateway] firewall driver: memory (dry-run)")
	} else {
		fw = firewall.NewNftablesDriver(nil)
		log.Printf("[gateway] firewall driver: nftables (inet inaply)")
	}

	edgeStore := edge.NewFileStore(*stateFile)
	store := session.NewStore(fw, edgeStore.Save)

	if snap, err := edgeStore.Load(); err == nil {
		if loadErr := store.LoadSnapshot(*snap); loadErr != nil {
			log.Printf("[gateway] failed restoring edge snapshot (%v), seeding demo property", loadErr)
			store.SeedDemoProperty()
		} else {
			log.Printf("[gateway] restored edge snapshot from %s", *stateFile)
		}
	} else {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("[gateway] could not read edge snapshot (%v), seeding demo property", err)
		} else {
			log.Printf("[gateway] fresh start: seeding demo hotel property (rooms 301, 302, 501)")
		}
		store.SeedDemoProperty()
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Janitor cleans up expired checkouts/sessions and rotates expired fallback PINs.
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				if expired := store.ExpireStale(now); expired > 0 {
					log.Printf("[janitor] cleared %d expired guest sessions", expired)
				}
			}
		}
	}()

	mdnsProxy := mdns.NewProxy(store)
	go func() {
		log.Printf("[mdns] proxy listening on %s", *mdnsAddr)
		if err := mdnsProxy.Serve(ctx, *mdnsAddr); err != nil && ctx.Err() == nil {
			log.Printf("[mdns] stopped with error: %v", err)
		}
	}()

	ssdpResp := ssdp.NewResponder(store)
	go func() {
		log.Printf("[ssdp] responder listening on %s", *ssdpAddr)
		if err := ssdpResp.Serve(ctx, *ssdpAddr); err != nil && ctx.Err() == nil {
			log.Printf("[ssdp] stopped with error: %v", err)
		}
	}()

	apiServer := api.NewServer(store, fw)
	if portalSecret := os.Getenv("INAPLY_PORTAL_SECRET"); portalSecret != "" {
		apiServer.SetPortalWebhookSecret(portalSecret)
	}
	if *assetsDir != "" {
		apiServer.SetAssetCache(assets.NewCache(*assetsDir, nil))
		log.Printf("[assets] serving local LAN cache from %s at /assets/", *assetsDir)
	}
	if *trustedProxies != "" {
		if err := apiServer.SetTrustedProxies(strings.Split(*trustedProxies, ",")); err != nil {
			log.Fatalf("[gateway] invalid -trusted-proxies: %v", err)
		}
	}
	httpSrv := &http.Server{
		Addr:              *httpAddr,
		Handler:           apiServer.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("[http] portal/API listening on %s (Portal: /portal, PIN: /pair, FO: /fo)", *httpAddr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[http] server failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("[gateway] shutting down gracefully...")
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutCtx)
}
