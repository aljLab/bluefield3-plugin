package utils

import (
	"fmt"
	"golang.zx2c4.com/wireguard/tun"
	"github.com/vishvananda/netlink"
	"github.com/mantra6g/bluefield3-plugin/pkg/proxy"
)



func CreateTunDevices(MTU int) (*proxy.TUN, *proxy.TUN, error) {
	tun0dev, err := tun.CreateTUN("tun0", MTU); if err != nil {
		return nil, nil, fmt.Errorf("failed to create tun device tun0: %w", err)
	}
	tun1dev, err := tun.CreateTUN("tun1", MTU); if err != nil {
		return nil, nil, fmt.Errorf("failed to create tun device tun1: %w", err)
	}

	tun0name, err := tun0dev.Name(); if err != nil {
		return nil, nil, fmt.Errorf("failed to get name of tun device tun0: %w", err)
	}
	
	tun1name, err := tun1dev.Name(); if err != nil {
		return nil, nil, fmt.Errorf("failed to get name of tun device tun1: %w", err)
	}

	tun0link, err := netlink.LinkByName(tun0name); if err != nil {
		return nil, nil, fmt.Errorf("failed to get interface for tun0: %w", err)
	}
	tun1link, err := netlink.LinkByName(tun1name); if err != nil {
		return nil, nil, fmt.Errorf("failed to get interface for tun1: %w", err)
	}

	err = netlink.LinkSetUp(tun0link); if err != nil {
		return nil, nil, fmt.Errorf("failed to set interface up: %w", err)
	}

	err = netlink.LinkSetUp(tun1link); if err != nil {
		return nil, nil, fmt.Errorf("failed to set interface up: %w", err)
	}

	tun0 := &proxy.TUN{
		Name: tun0name,
		Device: tun0dev,
		Link: tun0link,
	}

	tun1 := &proxy.TUN{
		Name: tun1name,
		Device: tun1dev,
		Link: tun1link,
	}

	return tun0, tun1, nil
}

func EnsureRule(family, priority, table int, iif string) error {
	rules, err := netlink.RuleList(family)
	if err != nil {
		return fmt.Errorf("list rules for family %d: %w", family, err)
	}

	// make idempotent
	for _, r := range rules {
		if r.Priority == priority &&
			r.Table == table &&
			r.IifName == iif {
			return nil
		}
	}

	r := netlink.NewRule()
	r.Family = family
	r.Priority = priority
	r.Table = table
	r.IifName = iif

	if err := netlink.RuleAdd(r); err != nil {
		return fmt.Errorf(
			"add rule priority=%d iif=%s lookup=%d: %w",
			priority, iif, table, err,
		)
	}

	return nil
}

func EnsureRoute(route *netlink.Route) error {
	if err := netlink.RouteReplace(route); err != nil {
		return fmt.Errorf(
			"replace route dst=%v table=%d: %w",
			route.Dst, route.Table, err,
		)
	}

	return nil
}