package main

import "testing"

func TestProbeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		listen  string
		want    string
		wantErr bool
	}{
		{listen: "127.0.0.1:26463", want: "http://127.0.0.1:26463/readyz"},
		{listen: "0.0.0.0:26463", want: "http://127.0.0.1:26463/readyz"},
		{listen: "[::]:26463", want: "http://127.0.0.1:26463/readyz"},
		{listen: ":26463", want: "http://127.0.0.1:26463/readyz"},
		{listen: "192.168.1.5:8080", want: "http://192.168.1.5:8080/readyz"},
		{listen: "no-port", wantErr: true},
	}
	for _, tt := range tests {
		got, err := probeURL(tt.listen)
		if tt.wantErr {
			if err == nil {
				t.Errorf("probeURL(%q): expected error, got %q", tt.listen, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("probeURL(%q): %v", tt.listen, err)
			continue
		}
		if got != tt.want {
			t.Errorf("probeURL(%q) = %q, want %q", tt.listen, got, tt.want)
		}
	}
}
