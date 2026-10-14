package bluefield3

import (
	"net"
)

type DualStackNetwork struct {
	IPv4Net *net.IPNet `json:"inet4,omitempty"`
	IPv6Net *net.IPNet `json:"inet6,omitempty"`
}

type Bluefield3Manager struct {
	DeviceId string
	oobIPs DualStackNetwork
}