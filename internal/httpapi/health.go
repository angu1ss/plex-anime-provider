package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const probeTimeout = 3 * time.Second

// ProbeFunc reports one readiness condition; nil means satisfied.
type ProbeFunc func(ctx context.Context) error

// Probes backs the Kubernetes-convention endpoints /livez and /readyz.
// Liveness is unconditional; readiness requires every registered
// condition to pass and the service not to be draining.
type Probes struct {
	mu           sync.RWMutex
	checks       []readinessCheck
	shuttingDown atomic.Bool
}

type readinessCheck struct {
	name  string
	probe ProbeFunc
}

// NewProbes returns an empty registry: live, ready, no conditions.
func NewProbes() *Probes { return &Probes{} }

// Register adds a named readiness condition; duplicate names panic.
func (p *Probes) Register(name string, probe ProbeFunc) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.checks {
		if c.name == name {
			panic(fmt.Sprintf("httpapi: readiness check %q registered twice", name))
		}
	}
	p.checks = append(p.checks, readinessCheck{name: name, probe: probe})
}

// StartShutdown makes /readyz fail so traffic drains away, while /livez
// keeps passing so the orchestrator waits out the drain.
func (p *Probes) StartShutdown() { p.shuttingDown.Store(true) }

func (p *Probes) handleLivez() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeText(w, http.StatusOK, "ok")
	}
}

// handleReadyz mirrors the kube-apiserver format: plain "ok" on success,
// per-check "[+]/[-]" lines with ?verbose or on any failure.
func (p *Probes) handleReadyz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), probeTimeout)
		defer cancel()

		var report strings.Builder
		ready := true

		if p.shuttingDown.Load() {
			ready = false
			report.WriteString("[-]shutdown: draining\n")
		}

		p.mu.RLock()
		checks := p.checks
		p.mu.RUnlock()
		for _, c := range checks {
			if err := c.probe(ctx); err != nil {
				ready = false
				fmt.Fprintf(&report, "[-]%s failed: %v\n", c.name, err)
			} else {
				fmt.Fprintf(&report, "[+]%s ok\n", c.name)
			}
		}

		switch {
		case ready && !r.URL.Query().Has("verbose"):
			writeText(w, http.StatusOK, "ok")
		case ready:
			writeText(w, http.StatusOK, report.String()+"readyz check passed")
		default:
			writeText(w, http.StatusServiceUnavailable, report.String()+"readyz check failed")
		}
	}
}

func writeText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
