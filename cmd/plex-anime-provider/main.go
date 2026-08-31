// Command plex-anime-provider implements the Plex custom metadata
// provider API for anime libraries. It listens on loopback by default:
// the provider API has no authentication, so wider exposure must be an
// explicit operator decision (see README).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/angu1ss/plex-anime-provider/internal/httpapi"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

const (
	defaultListen   = "127.0.0.1:26463"
	envListen       = "PLEX_ANIME_PROVIDER_LISTEN"
	shutdownTimeout = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	listen := flag.String("listen", envOr(envListen, defaultListen),
		"listen address host:port (env "+envListen+")")
	healthcheck := flag.Bool("healthcheck", false,
		"probe the running instance's /readyz and exit; used as container HEALTHCHECK")
	flag.Parse()

	if *healthcheck {
		return selfCheck(*listen)
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	probes := httpapi.NewProbes()
	srv := &http.Server{
		Addr:              *listen,
		Handler:           httpapi.NewRouter(version, probes),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("server starting", "addr", *listen, "version", version)
		errCh <- srv.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		return err // ListenAndServe never returns nil
	case <-ctx.Done():
	}

	// Drain: fail /readyz so traffic moves away, then shut down.
	probes.StartShutdown()
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Info("server stopped")
	return nil
}

// selfCheck probes the running instance's /readyz; distroless images have
// no shell or curl, so the binary probes itself.
func selfCheck(listen string) error {
	url, err := probeURL(listen)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("readyz: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

// probeURL maps a listen address to its /readyz URL, substituting
// loopback for wildcard hosts.
func probeURL(listen string) (string, error) {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return "", fmt.Errorf("listen address %q: %w", listen, err)
	}
	switch host {
	case "", "0.0.0.0", "::":
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + "/readyz", nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
