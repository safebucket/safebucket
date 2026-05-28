package configuration

import (
	"fmt"
	"net"
	"strings"
)

func ParseTrustedProxies(entries []string) ([]*net.IPNet, error) {
	nets := make([]*net.IPNet, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		if _, network, err := net.ParseCIDR(entry); err == nil {
			nets = append(nets, network)
			continue
		}

		ip := net.ParseIP(entry)
		if ip == nil {
			return nil, fmt.Errorf("invalid trusted proxy entry %q: not a valid IP or CIDR", entry)
		}

		bits := net.IPv4len * 8
		if ip.To4() == nil {
			bits = net.IPv6len * 8
		}
		nets = append(nets, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
	}

	return nets, nil
}
