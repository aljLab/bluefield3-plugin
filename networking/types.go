package networking

import (
	"net"
)

type P4TargetConfigRequest struct {
	TargetName string
}

type DualStackNetwork struct {
	IPv4Net *net.IPNet `json:"inet4,omitempty"`
	IPv6Net *net.IPNet `json:"inet6,omitempty"`
}

type DualStackAddress struct {
	IPv4 net.IP `json:"ip4,omitempty"`
	IPv6 net.IP `json:"ip6,omitempty"`
}

type TargetNetConfig struct {
	NfCIDR string
}

type NetworkConfig struct {
	IPNets       DualStackNetwork `json:"ip_nets"` // assigned to iml0
	ClusterCIDRs DualStackNetwork `json:"cluster_cidrs"`
	ServiceCIDRs DualStackNetwork `json:"service_cidrs"`
	Gateways     DualStackAddress `json:"gateways"`
	IfaceName    string           `json:"iface_name"`  // veth iface name on host bridge <--> iml0
	BridgeName   string           `json:"bridge_name"` // host bridge
	MTU          uint32           `json:"mtu"`
}
