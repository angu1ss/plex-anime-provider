// Package httpapi wires the HTTP surface of the provider.
package httpapi

import (
	"encoding/json"
	"net/http"
)

// NewRouter returns the root handler of the service.
func NewRouter(version string, probes *Probes) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /health", handleHealth(version))
	mux.Handle("GET /livez", probes.handleLivez())
	mux.Handle("GET /healthz", probes.handleLivez())
	mux.Handle("GET /readyz", probes.handleReadyz())
	return mux
}

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func handleHealth(version string) http.HandlerFunc {
	// The response is immutable, so encode it once.
	body, err := json.Marshal(healthResponse{Status: "ok", Version: version})
	if err != nil {
		panic(err)
	}
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}
}
