package p4target

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1alpha1 "github.com/mantra6g/iml/api/core/v1alpha1"
	p4v1 "github.com/p4lang/p4runtime/go/p4/v1"
	/* "github.com/vishvananda/netlink"
	"golang.org/x/sys/unix" */
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	net_utils "github.com/mantra6g/bluefield3-plugin/networking"
	// "github.com/mantra6g/bluefield3-plugin/api/v1alpha1"
	"github.com/mantra6g/bluefield3-plugin/pkg/ipam"
	/* "github.com/mantra6g/bluefield3-plugin/pkg/proxy"
	"github.com/mantra6g/bluefield3-plugin/pkg/proxy/utils" */
)

type Condition struct {
	Status  metav1.ConditionStatus
	Reason  string
	Message string
}

type ManagerConfig struct {
	Name       string
	TargetIPs  []net.IP
	DriverIPs  []net.IP
	MaxNFSlots int
	P4Client   p4v1.P4RuntimeClient
	BridgeName string
	Log        logr.Logger
}


type Manager interface {
	GetName() string
	GetCapacity() corev1.ResourceList
	GetAllocatable() corev1.ResourceList
	GetHealthyCondition() corev1alpha1.P4TargetCondition
	GetReadyCondition(corev1alpha1.P4TargetCondition, corev1alpha1.P4TargetCondition) corev1alpha1.P4TargetCondition
	GetNetworkConfiguredCondition() corev1alpha1.P4TargetCondition
	GetOccupiedCondition() corev1alpha1.P4TargetCondition
	GetTargetIPs() []net.IP
	GetDriverIPs() []net.IP
	EnsureNetworkConfiguration(net_utils.TargetNetConfig) error
	AllocateNetworkFunctionIP() (netip.Addr, error)
	GetNfCIDR() netip.Prefix
}

func NewManager(cfg ManagerConfig) (Manager, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("manager name is required")
	}
	if cfg.MaxNFSlots <= 0 {
		return nil, fmt.Errorf("max NF slots must be > 0")
	}
	if cfg.P4Client == nil {
		return nil, fmt.Errorf("P4Runtime client is required")
	}
	return &RealManager{
		name:         cfg.Name,
		targetIPs:    cfg.TargetIPs,
		driverIPs:    cfg.DriverIPs,
		maxNFSlots:   cfg.MaxNFSlots,
		p4client:     cfg.P4Client,
		// BridgeName:   cfg.BridgeName,
		allocatedIPs: make(map[netip.Addr]struct{}),
		log:          cfg.Log,
	}, nil
}

// Compile-time assertion to ensure RealManager implements the Manager interface
var _ Manager = &RealManager{}

type RealManager struct {
	name       string
	targetIPs  []net.IP
	driverIPs  []net.IP
	maxNFSlots int
	p4client   p4v1.P4RuntimeClient
	// BridgeName string // created from cni
	log        logr.Logger

	mu           sync.RWMutex
	cidr         netip.Prefix
	ipAllocator  *ipam.AddrAllocator
	allocatedIPs map[netip.Addr]struct{}
	proxyCancel  context.CancelFunc
}

func (r *RealManager) GetNfCIDR() netip.Prefix {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cidr
}

func (r *RealManager) GetName() string { return r.name }

func (r *RealManager) GetTargetIPs() []net.IP { return r.targetIPs }

func (r *RealManager) GetDriverIPs() []net.IP { return r.driverIPs }

func (r *RealManager) EnsureNetworkConfiguration(cfg net_utils.TargetNetConfig) error {
	cidr, err := netip.ParsePrefix(cfg.NfCIDR)
	if err != nil {
		return fmt.Errorf("invalid CIDR %q: %w", cfg.NfCIDR, err)
	}
	cidr = cidr.Masked()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cidr == cidr {
		return nil
	}
	
	alloc, err := ipam.NewAddrAllocator(cidr)
	if err != nil {
		return fmt.Errorf("failed to create IP allocator: %w", err)
	}
	r.cidr = cidr
	r.ipAllocator = alloc
	r.allocatedIPs = make(map[netip.Addr]struct{})
	
	/* // TODO: get actual MTU
	MTU := 1294
	table1 := 200
	prio1 := 200
	// ip -6 route <netfunc.AssignedIP> dev tun0 -> configured by NF manager but tun devices need to exist first
	tun0, tun1, err := utils.CreateTunDevices(MTU); if err != nil {
		return fmt.Errorf("failed to create tun devices: %w", err)
	}

	inIfaceName := "bf0" // todo get from network annotations or owner BlueField3Target
	outIfaceName := "bf1"

	inIface, err := netlink.LinkByName(inIfaceName); if err != nil {
		return fmt.Errorf("failed to get incoming interface: %w", err)
	}
	outIface, err := netlink.LinkByName(outIfaceName); if err != nil {
		return fmt.Errorf("failed to get incoming interface: %w", err)
	}

	// ip (-6) rule iif tun0 lookup 200 -> v4/6 200: default via <bogus-ip> dev bf0 (get name from Pod spec)
	err = utils.EnsureRule(netlink.FAMILY_V4, prio1, table1, tun0.Name); if err != nil {
		return fmt.Errorf("failed adding rule: %d, %d, %d, %s", 
		netlink.FAMILY_V4, prio1, table1, tun0.Name)
	}
	err = utils.EnsureRule(netlink.FAMILY_V6, prio1, table1, tun0.Name); if err != nil {
		return fmt.Errorf("failed adding rule: %d, %d, %d, %s", 
		netlink.FAMILY_V6, prio1, table1, tun0.Name)
	}

	bogusGW4 := net.IPv4(10, 13, 37, 1)
	bogusGW6, err := r.AllocateNetworkFunctionIP(); if err != nil {
		return fmt.Errorf("failed to allocate IPv6 ip for bogus gateway")
	}
	_, defaultDst4, _ := net.ParseCIDR("0.0.0.0/0")
	_, defaultDst6, _ := net.ParseCIDR("::/0")
	
	route4 := &netlink.Route{
		LinkIndex: inIface.Attrs().Index,
		Gw:        bogusGW4,
		Dst:       defaultDst4,
		Table:     table1,
		Family:    netlink.FAMILY_V4,
	}

	route6 := &netlink.Route{
		LinkIndex: inIface.Attrs().Index,
		Gw:        net.ParseIP(bogusGW6.String()),
		Dst:       defaultDst6,
		Table:     table1,
		Family:    netlink.FAMILY_V6,
	}

	err = utils.EnsureRoute(route4); if err != nil {
		return fmt.Errorf("failed adding route -%d %s via %s dev %s table %d", 
		netlink.FAMILY_V4, defaultDst4.String(), bogusGW4.String(), inIfaceName, table1)
	}
	err = utils.EnsureRoute(route6); if err != nil {
		return fmt.Errorf("failed adding route -%d %s via %s dev %s table %d", 
		netlink.FAMILY_V6, defaultDst6.String(), bogusGW6.String(), inIfaceName, table1)
	}

	// ip neigh <bogus-ip> lladdr <bf1-mac> nud permanent
	hwAddr := outIface.Attrs().HardwareAddr
	neigh4 := &netlink.Neigh{
        LinkIndex:    inIface.Attrs().Index,
        IP:           bogusGW4,
        HardwareAddr: hwAddr,
        State:        netlink.NUD_PERMANENT,
    }
	netlink.NeighSet(neigh4)

	neigh6 := &netlink.Neigh{
        LinkIndex:    inIface.Attrs().Index,
        IP:           net.ParseIP(bogusGW6.String()),
        HardwareAddr: hwAddr,
        State:        netlink.NUD_PERMANENT,
    }
	netlink.NeighSet(neigh6)

	table2 := 201
	prio2 := 201

	// ip (-6) rule iif bf1 lookup 201: v4/6 201: default dev tun1
	err = utils.EnsureRule(4, prio2, table2, inIfaceName); if err != nil {
		return fmt.Errorf("failed adding rule: family IPv%d, priority %d, table %d, iif %s", 
		netlink.FAMILY_V4, prio2, table2, inIfaceName)
	}
	err = utils.EnsureRule(6, prio2, table2, inIfaceName); if err != nil {
		return fmt.Errorf("failed adding rule: family IPv%d, priority %d, table %d, iif %s", 
		netlink.FAMILY_V6, prio2, table2, inIfaceName)
	}

	err = utils.EnsureRoute(&netlink.Route{
		LinkIndex: tun1.Link.Attrs().Index,
		Dst: defaultDst4,
		Table: table2,
		Family: netlink.FAMILY_V4,
	}); if err != nil {
		return fmt.Errorf("failed to add route -%d %s dev %s table %d", 
		netlink.FAMILY_V4, defaultDst4.String(), tun1.Name, table2)
	}

	err = utils.EnsureRoute(&netlink.Route{
		LinkIndex: tun1.Link.Attrs().Index,
		Dst: defaultDst6,
		Table: table2,
		Family: netlink.FAMILY_V6,
	}); if err != nil {
		return fmt.Errorf("failed to add route -%d %s dev %s table %d", 
		netlink.FAMILY_V6, defaultDst6.String(), tun1.Name, table2)
	}

	// ip -6 route default <gateway-ip> (get from Pod spec) via iml0
	defaultIface, err := netlink.LinkByName("iml0"); if err != nil {
		return fmt.Errorf("failed to get iml0 interface: %w", err)
	}
	err = utils.EnsureRoute(&netlink.Route{
		LinkIndex: defaultIface.Attrs().Index,
		Dst: defaultDst6,
		//Gw: IpOfRoutingBridge,
		Table: unix.RT_TABLE_MAIN,
		Family: netlink.FAMILY_V6,
	})

	cache := &proxy.SRHCache{}

	decapWorker := &proxy.SRv6ProxyWorker{
		IfaceName: tun0.Name,
		TunDev:    tun0.Device,
		Cache:     cache,
	}
	encapWorker := &proxy.SRv6ProxyWorker{
		IfaceName: tun1.Name,
		TunDev:    tun1.Device,
		Cache:     cache,
	}

	proxyCancel, err := r.ensureProxyRunning(context.Background(), decapWorker, encapWorker); if err != nil {
		return fmt.Errorf("Couldn't start proxy background workers: %w", err)
	}

	r.proxyCancel = proxyCancel */
	return nil
}

func (r *RealManager) AllocateNetworkFunctionIP() (netip.Addr, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ipAllocator == nil {
		return netip.Addr{}, fmt.Errorf("network not configured: call EnsureNetworkConfiguration first")
	}
	addr, err := r.ipAllocator.Next()
	if err != nil {
		return netip.Addr{}, err
	}
	r.allocatedIPs[addr] = struct{}{}
	return addr, nil
}

func (r *RealManager) GetCapacity() corev1.ResourceList {
	capacity := corev1.ResourceList{
		ResourceNFSlots: *resource.NewQuantity(int64(r.maxNFSlots), resource.DecimalSI),
	}
	if tableCapacity := r.queryTableCapacity(); tableCapacity > 0 {
		capacity[ResourceTableEntries] = *resource.NewQuantity(tableCapacity, resource.DecimalSI)
	}
	return capacity
}

func (r *RealManager) GetAllocatable() corev1.ResourceList {
	r.mu.RLock()
	usedSlots := int64(len(r.allocatedIPs))
	r.mu.RUnlock()

	freeSlots := max(int64(r.maxNFSlots)-usedSlots, 0)
	allocatable := corev1.ResourceList{
		ResourceNFSlots: *resource.NewQuantity(freeSlots, resource.DecimalSI),
	}
	tableCapacity := r.queryTableCapacity()
	if tableCapacity > 0 {
		allocatable[ResourceTableEntries] = *resource.NewQuantity(max(tableCapacity-r.queryTableUsed(), 0), resource.DecimalSI)
	}
	return allocatable
}

func (r *RealManager) queryTableCapacity() int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := r.p4client.GetForwardingPipelineConfig(ctx, &p4v1.GetForwardingPipelineConfigRequest{
		ResponseType: p4v1.GetForwardingPipelineConfigRequest_P4INFO_AND_COOKIE,
	})
	if err != nil || resp.Config == nil || resp.Config.P4Info == nil {
		return 0
	}
	var total int64
	for _, t := range resp.Config.P4Info.Tables {
		total += t.Size
	}
	return total
}

func (r *RealManager) queryTableUsed() int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := r.p4client.Read(ctx, &p4v1.ReadRequest{
		Entities: []*p4v1.Entity{{Entity: &p4v1.Entity_TableEntry{TableEntry: &p4v1.TableEntry{}}}},
	})
	if err != nil {
		return 0
	}
	var count int64
	for {
		resp, err := stream.Recv()
		if err != nil {
			break
		}
		count += int64(len(resp.Entities))
	}
	return count
}

func (r *RealManager) GetHealthyCondition() corev1alpha1.P4TargetCondition {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	now := metav1.NewTime(time.Now())
	_, err := r.p4client.Capabilities(ctx, &p4v1.CapabilitiesRequest{})
	if err != nil {
		return corev1alpha1.P4TargetCondition{
			Type:              ConditionHealthy,
			Status:            metav1.ConditionFalse,
			LastHeartbeatTime: now,
			Reason:            "TargetUnreachable",
			Message:           err.Error(),
		}
	}
	return corev1alpha1.P4TargetCondition{
		Type:              ConditionHealthy,
		Status:            metav1.ConditionTrue,
		LastHeartbeatTime: now,
		Reason:            "TargetReachable",
		Message:           "The DPL Service on the BlueField 3 is reachable",
	}
}

func (r *RealManager) GetReadyCondition(healthyCond corev1alpha1.P4TargetCondition,
	netconfCond corev1alpha1.P4TargetCondition) corev1alpha1.P4TargetCondition {
	if healthyCond.Status == metav1.ConditionTrue && netconfCond.Status == metav1.ConditionTrue {
		return corev1alpha1.P4TargetCondition{
			Type:              corev1alpha1.P4TargetConditionReady,
			Status:            metav1.ConditionTrue,
			LastHeartbeatTime: metav1.NewTime(time.Now()),
			Reason:            "TargetReady",
			Message:           "The target is ready.",
		}
	}
	return corev1alpha1.P4TargetCondition{
		Type:              corev1alpha1.P4TargetConditionReady,
		Status:            metav1.ConditionFalse,
		LastHeartbeatTime: metav1.NewTime(time.Now()),
		Reason:            "TargetNotReady",
		Message:           "The target is not ready.",
	}
}

func (r *RealManager) GetNetworkConfiguredCondition() corev1alpha1.P4TargetCondition {
	r.mu.RLock()
	cidr := r.cidr
	r.mu.RUnlock()
	now := metav1.NewTime(time.Now())
	if !cidr.IsValid() {
		return corev1alpha1.P4TargetCondition{
			Type:              ConditionNetworkConfigured,
			Status:            metav1.ConditionFalse,
			LastHeartbeatTime: now,
			Reason:            "NoCIDR",
			Message:           "No CIDR was assigned to the P4Target yet.",
		}
	}
	return corev1alpha1.P4TargetCondition{
		Type:              ConditionNetworkConfigured,
		Status:            metav1.ConditionTrue,
		LastHeartbeatTime: now,
		Reason:            "CIDRConfigured",
		Message:           cidr.String(),
	}
}

func (r *RealManager) GetOccupiedCondition() corev1alpha1.P4TargetCondition {
	r.mu.RLock()
	used := len(r.allocatedIPs)
	r.mu.RUnlock()
	now := metav1.NewTime(time.Now())
	if used >= r.maxNFSlots {
		return corev1alpha1.P4TargetCondition{
			Type:              ConditionOccupied,
			Status:            metav1.ConditionTrue,
			LastHeartbeatTime: now,
			Reason:            "NoSlotsAvailable",
			Message:           fmt.Sprintf("%d/%d slots used", used, r.maxNFSlots),
		}
	}
	return corev1alpha1.P4TargetCondition{
		Type:              ConditionOccupied,
		Status:            metav1.ConditionFalse,
		LastHeartbeatTime: now,
		Reason:            "SlotsAvailable",
		Message:           fmt.Sprintf("%d/%d slots used", used, r.maxNFSlots),
	}
}

/* func (r *RealManager) ensureProxyRunning(ctx context.Context, w1, w2 *proxy.SRv6ProxyWorker) (context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(ctx)

	go w1.DecapTraffic(ctx, r.log)

	go w2.EncapTraffic(ctx, r.log)

	return cancel, nil
}
 */