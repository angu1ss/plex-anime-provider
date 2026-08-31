package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func probeGet(t *testing.T, probes *Probes, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	NewRouter("test", probes).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestReadyzNoChecks(t *testing.T) {
	t.Parallel()

	rec := probeGet(t, NewProbes(), "/readyz")
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("readyz = %d %q, want 200 %q", rec.Code, rec.Body.String(), "ok")
	}
}

func TestReadyzVerbose(t *testing.T) {
	t.Parallel()

	probes := NewProbes()
	probes.Register("first", func(context.Context) error { return nil })
	probes.Register("second", func(context.Context) error { return nil })

	rec := probeGet(t, probes, "/readyz?verbose")
	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, body)
	}
	for _, want := range []string{"[+]first ok", "[+]second ok", "readyz check passed"} {
		if !strings.Contains(body, want) {
			t.Errorf("body %q missing %q", body, want)
		}
	}
}

func TestReadyzFailingCheck(t *testing.T) {
	t.Parallel()

	probes := NewProbes()
	probes.Register("good", func(context.Context) error { return nil })
	probes.Register("bad", func(context.Context) error { return errors.New("boom") })

	rec := probeGet(t, probes, "/readyz")
	body := rec.Body.String()
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body: %s", rec.Code, body)
	}
	for _, want := range []string{"[+]good ok", "[-]bad failed: boom", "readyz check failed"} {
		if !strings.Contains(body, want) {
			t.Errorf("body %q missing %q", body, want)
		}
	}
}

func TestReadyzShutdownDrain(t *testing.T) {
	t.Parallel()

	probes := NewProbes()
	probes.StartShutdown()

	if rec := probeGet(t, probes, "/readyz"); rec.Code != http.StatusServiceUnavailable ||
		!strings.Contains(rec.Body.String(), "[-]shutdown: draining") {
		t.Fatalf("readyz after shutdown = %d %q, want 503 with drain marker", rec.Code, rec.Body.String())
	}
	// Liveness must keep succeeding during the drain.
	for _, path := range []string{"/livez", "/healthz"} {
		if rec := probeGet(t, probes, path); rec.Code != http.StatusOK {
			t.Errorf("%s during shutdown = %d, want 200", path, rec.Code)
		}
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("duplicate Register did not panic")
		}
	}()
	probes := NewProbes()
	probes.Register("dup", func(context.Context) error { return nil })
	probes.Register("dup", func(context.Context) error { return nil })
}
