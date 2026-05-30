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

		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy entry %q: must be CIDR notation (e.g. 10.0.0.0/8)", entry)
		}
		nets = append(nets, network)
	}

	return nets, nil
}
