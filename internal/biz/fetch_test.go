package biz

import "testing"

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid http", "http://example.com/debug/pprof/heap", false},
		{"valid https", "https://example.com/profile", false},
		{"file scheme", "file:///etc/passwd", true},
		{"no scheme", "example.com/path", true},
		{"ftp scheme", "ftp://example.com/file", true},
		{"loopback ipv4", "http://127.0.0.1:8080/heap", true},
		{"loopback ipv6", "http://[::1]:8080/heap", true},
		{"private 10.x", "http://10.0.0.1/heap", true},
		{"private 192.168.x", "http://192.168.1.1/heap", true},
		{"empty url", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}
