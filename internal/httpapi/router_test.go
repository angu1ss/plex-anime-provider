package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{"health ok", http.MethodGet, "/health", http.StatusOK},
		{"livez ok", http.MethodGet, "/livez", http.StatusOK},
		{"healthz alias ok", http.MethodGet, "/healthz", http.StatusOK},
		{"readyz ok", http.MethodGet, "/readyz", http.StatusOK},
		{"health wrong method", http.MethodPost, "/health", http.StatusMethodNotAllowed},
		{"unknown route", http.MethodGet, "/nope", http.StatusNotFound},
	}

	router := NewRouter("test-version", NewProbes())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("%s %s: status = %d, want %d", tt.method, tt.path, rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHealthBody(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	NewRouter("v1.2.3", NewProbes()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if got, want := rec.Header().Get("Content-Type"), "application/json"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}

	var body healthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if want := (healthResponse{Status: "ok", Version: "v1.2.3"}); body != want {
		t.Errorf("body = %+v, want %+v", body, want)
	}
}
