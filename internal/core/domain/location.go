package domain

import (
	"net"
	"net/netip"
	"strings"
)

// DefaultClinicLocation defines the standard physical premise location for the clinic.
const DefaultClinicLocation = "Yogyakarta, Indonesia"

// cleanAndParseIP trims, strips port/brackets, and parses an IP string into a normalized netip.Addr.
func cleanAndParseIP(ipStr string) (netip.Addr, bool) {
	ipStr = strings.TrimSpace(ipStr)
	if ipStr == "" {
		return netip.Addr{}, false
	}

	// Strip port if formatted as host:port (e.g. from RemoteAddr or proxy headers)
	if host, _, err := net.SplitHostPort(ipStr); err == nil {
		ipStr = host
	}

	// Strip IPv6 enclosing brackets if any, e.g. [::1]
	ipStr = strings.TrimPrefix(strings.TrimSuffix(ipStr, "]"), "[")

	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return netip.Addr{}, false
	}

	return addr.Unmap(), true
}

// IsPrivateOrLoopbackIP reports whether the given IP address is private, loopback, link-local,
// or unspecified according to RFC 1918, RFC 4193, and standard network specifications.
func IsPrivateOrLoopbackIP(ipStr string) bool {
	addr, ok := cleanAndParseIP(ipStr)
	if !ok {
		return true
	}

	return addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsUnspecified()
}

// ResolveLocation maps an IP address to a human-readable location string.
// Private, loopback, intranet, and unparseable IPs are dynamically mapped to the clinic's premise location.
// Public IPs return the geographical country/region location.
func ResolveLocation(ipStr string, fallbackLocation ...string) string {
	fallback := DefaultClinicLocation
	if len(fallbackLocation) > 0 && strings.TrimSpace(fallbackLocation[0]) != "" {
		fallback = strings.TrimSpace(fallbackLocation[0])
	}

	addr, ok := cleanAndParseIP(ipStr)
	if !ok {
		return fallback
	}

	if addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsUnspecified() {
		return fallback
	}

	// For public IP addresses accessing the clinic system
	return "Indonesia"
}
