package middleware

import (
	"net/http"
	"testing"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		realIP     string
		forwarded  string
		remoteAddr string
		want       string
	}{
		{
			name:       "X-Real-IP wins over X-Forwarded-For and RemoteAddr",
			realIP:     "1.2.3.4",
			forwarded:  "5.6.7.8",
			remoteAddr: "9.10.11.12:5000",
			want:       "1.2.3.4",
		},
		{
			name:       "X-Forwarded-For uses rightmost when X-Real-IP absent",
			forwarded:  "1.1.1.1, 2.2.2.2, 3.3.3.3",
			remoteAddr: "9.10.11.12:5000",
			want:       "3.3.3.3",
		},
		{
			name:       "RemoteAddr fallback strips port",
			remoteAddr: "9.10.11.12:5000",
			want:       "9.10.11.12",
		},
		{
			name:       "RemoteAddr without port returned as-is",
			remoteAddr: "9.10.11.12",
			want:       "9.10.11.12",
		},
		{
			name:       "X-Real-IP gets whitespace trimmed",
			realIP:     "  1.2.3.4  ",
			remoteAddr: "9.10.11.12:5000",
			want:       "1.2.3.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Header:     http.Header{},
				RemoteAddr: tt.remoteAddr,
			}
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}
			if tt.forwarded != "" {
				req.Header.Set("X-Forwarded-For", tt.forwarded)
			}

			got := clientIP(req)
			if got != tt.want {
				t.Errorf("clientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}
