package proxy

import(
	"sync"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/tun"
)

type SRv6Header struct {
	Start           [24]byte
	DstAddr         [16]byte
	NextHeader      [1]byte
	HdrExtLen       [1]byte
	RoutingType     [1]byte
	SegmentsLeft    [1]byte
	FirstSeg        [1]byte
	Flags           [1]byte
	Tag             [2]byte
	SegmentList     []byte
	Tlvs            []byte
}

type SRHCache struct {
	Header      SRv6Header
	TotalSize   int
	Raw         []byte
	Mu          sync.Mutex
}

type SRv6ProxyWorker struct {
	IfaceName string
	TunDev    tun.Device
	Cache     *SRHCache
}

type TUN struct {
	Name string
	Device tun.Device
	Link netlink.Link
}