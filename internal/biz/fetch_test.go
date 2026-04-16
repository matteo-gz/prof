package biz

import "testing"

func TestValidateURL_DenyPrivateIP(t *testing.T) {
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
			err := validateURL(tt.url, true)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateURL(%q, true) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidateURL_AllowPrivateIP(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"loopback allowed", "http://127.0.0.1:8080/heap", false},
		{"private 10.x allowed", "http://10.0.0.1/heap", false},
		{"private 192.168.x allowed", "http://192.168.1.1/heap", false},
		{"file scheme still blocked", "file:///etc/passwd", true},
		{"no scheme still blocked", "example.com/path", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.url, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateURL(%q, false) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}
