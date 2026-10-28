package util

import (
	"fmt"
	"net"
	"strings"
)

// ParseTrustedProxies parses a comma-separated list of proxy IPs or CIDRs.
// An empty value means no proxy is trusted, so X-Forwarded-For is ignored and
// client addresses come from the socket peer.
func ParseTrustedProxies(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var proxies []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if net.ParseIP(part) == nil {
			if _, _, err := net.ParseCIDR(part); err != nil {
				return nil, fmt.Errorf("invalid trusted proxy %q: must be an IP or CIDR", part)
			}
		}
		proxies = append(proxies, part)
	}

	return proxies, nil
}
