package configuration

import (
	"net"
	"strings"

	"go.uber.org/zap"
)

func ParseTrustedProxies(entries []string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			zap.L().Fatal("Invalid trusted_proxies configuration",
				zap.String("entry", entry), zap.Error(err))
		}
		nets = append(nets, network)
	}

	return nets
}
