package domain

import (
	"testing"
)

func TestIsPrivateOrLoopbackIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{name: "empty string", ip: "", expected: true},
		{name: "whitespace only", ip: "   ", expected: true},
		{name: "invalid IP string", ip: "invalid-ip-addr", expected: true},
		{name: "IPv4 loopback 127.0.0.1", ip: "127.0.0.1", expected: true},
		{name: "IPv4 loopback with port 127.0.0.1:8080", ip: "127.0.0.1:8080", expected: true},
		{name: "IPv4 loopback subnet 127.0.0.53", ip: "127.0.0.53", expected: true},
		{name: "IPv6 loopback ::1", ip: "::1", expected: true},
		{name: "IPv6 loopback with brackets [::1]", ip: "[::1]", expected: true},
		{name: "IPv6 loopback with port [::1]:8080", ip: "[::1]:8080", expected: true},
		{name: "IPv4-mapped IPv6 loopback ::ffff:127.0.0.1", ip: "::ffff:127.0.0.1", expected: true},
		{name: "IPv4 private 10.0.0.1", ip: "10.0.0.1", expected: true},
		{name: "IPv4 private with port 10.0.0.1:9000", ip: "10.0.0.1:9000", expected: true},
		{name: "IPv4 private 172.16.0.5", ip: "172.16.0.5", expected: true},
		{name: "IPv4 private 192.168.1.100", ip: "192.168.1.100", expected: true},
		{name: "IPv4 link-local 169.254.1.1", ip: "169.254.1.1", expected: true},
		{name: "IPv4 unspecified 0.0.0.0", ip: "0.0.0.0", expected: true},
		{name: "IPv6 link-local fe80::1", ip: "fe80::1", expected: true},
		{name: "Public IPv4 8.8.8.8", ip: "8.8.8.8", expected: false},
		{name: "Public IPv4 with port 8.8.8.8:443", ip: "8.8.8.8:443", expected: false},
		{name: "Public IPv4 114.122.10.5", ip: "114.122.10.5", expected: false},
		{name: "Public IPv6 2001:4860:4860::8888", ip: "2001:4860:4860::8888", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPrivateOrLoopbackIP(tt.ip)
			if got != tt.expected {
				t.Errorf("IsPrivateOrLoopbackIP(%q) = %v, expected %v", tt.ip, got, tt.expected)
			}
		})
	}
}

func TestResolveLocation(t *testing.T) {
	tests := []struct {
		name      string
		ip        string
		fallbacks []string
		expected  string
	}{
		{
			name:      "empty IP defaults to Yogyakarta",
			ip:        "",
			fallbacks: nil,
			expected:  "Yogyakarta, Indonesia",
		},
		{
			name:      "whitespace IP defaults to custom fallback",
			ip:        "   ",
			fallbacks: []string{"Surakarta, Indonesia"},
			expected:  "Surakarta, Indonesia",
		},
		{
			name:      "empty custom fallback string uses DefaultClinicLocation",
			ip:        "127.0.0.1",
			fallbacks: []string{""},
			expected:  "Yogyakarta, Indonesia",
		},
		{
			name:      "invalid IP uses fallback",
			ip:        "not-an-ip",
			fallbacks: []string{"Bandung, Indonesia"},
			expected:  "Bandung, Indonesia",
		},
		{
			name:      "localhost 127.0.0.1 uses custom fallback",
			ip:        "127.0.0.1",
			fallbacks: []string{"Sleman, Indonesia"},
			expected:  "Sleman, Indonesia",
		},
		{
			name:      "localhost with port 127.0.0.1:8080 uses custom fallback",
			ip:        "127.0.0.1:8080",
			fallbacks: []string{"Sleman, Indonesia"},
			expected:  "Sleman, Indonesia",
		},
		{
			name:      "private IP 192.168.1.1 uses DefaultClinicLocation",
			ip:        "192.168.1.1",
			fallbacks: nil,
			expected:  "Yogyakarta, Indonesia",
		},
		{
			name:      "public IP returns country location",
			ip:        "114.122.10.5",
			fallbacks: nil,
			expected:  "Indonesia",
		},
		{
			name:      "public IP with port returns country location",
			ip:        "114.122.10.5:443",
			fallbacks: nil,
			expected:  "Indonesia",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveLocation(tt.ip, tt.fallbacks...)
			if got != tt.expected {
				t.Errorf("ResolveLocation(%q, %v) = %q, expected %q", tt.ip, tt.fallbacks, got, tt.expected)
			}
		})
	}
}
