package common

import (
	"net"
	"testing"
)

func TestIsPrivateIPRecognizesSpecialRanges(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{name: "carrier grade nat", ip: "100.64.0.1", want: true},
		{name: "test net 1", ip: "192.0.2.1", want: true},
		{name: "benchmark net", ip: "198.18.0.1", want: true},
		{name: "test net 3", ip: "203.0.113.10", want: true},
		{name: "unspecified ipv4", ip: "0.0.0.0", want: true},
		{name: "documentation ipv6", ip: "2001:db8::1", want: true},
		{name: "unique local ipv6", ip: "fc00::1", want: true},
		{name: "ipv4 mapped ipv6", ip: "::ffff:127.0.0.1", want: true},
		{name: "public ipv4", ip: "8.8.8.8", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := net.ParseIP(tt.ip)
			if parsed == nil {
				t.Fatalf("failed to parse ip %q", tt.ip)
			}

			if got := isPrivateIP(parsed); got != tt.want {
				t.Fatalf("isPrivateIP(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}
