package proxy

import (
	"bytes"
	"context"
	"fmt"

	"github.com/go-logr/logr"
)

const(
	VIRTIO_OFFSET=16
	BUFF_SIZE = 65535 //64Kib
	BUFF_COUNT = 1 // worker.TunDev.BatchSize()
)

func (worker *SRv6ProxyWorker) DecapTraffic(ctx context.Context, log logr.Logger) {
	r_offset := VIRTIO_OFFSET // make at least room for virtio linux header
	buffSize := BUFF_SIZE
	buffCount := BUFF_COUNT

	// create one buffer for each (potential) packet, each with offset + max package size (also for writing, GRO)
	// reuse buffers to avoid alloc + copy for each packet
	bufs := make([][]byte, buffCount)
	for i := range bufs {
		bufs[i] = make([]byte, buffSize)
	}
	sizes := make([]int, buffCount)

	for {
		// RESET 
		// reset write offset
		w_offset := r_offset
		// resize reused write buffers (slices) to original capacity for reading
		for i := range bufs {
			bufs[i] = bufs[i][:cap(bufs[i])]
		}


		n, err := worker.TunDev.Read(bufs, sizes, r_offset)
		if err != nil {
			log.Info("SRv6 Proxy: TUN read failed", "interface", worker.IfaceName, "error", err)
			return
		}

		// n is usually 1, since we read packets 1-by-1
		for i := range n {
			packet := bufs[i][r_offset:r_offset + sizes[i]]
			if len(packet) == 0 {
				continue
			}

			version := packet[0] >> 4

			if version == 6 {
				nextHeader := packet[6]
				// if IPv6 Routing Header
				if nextHeader == 43 {
					// if RoutingType == SRv6
					if packet[42:43][0] == 4 {
						totalHeaderBytes, err := worker.processSRv6Header(packet); if err != nil {
							fmt.Printf("Encountered error when performing decapsulation: %s.\n Skipping.", err)
							continue
						}

						w_offset = w_offset + int(totalHeaderBytes) // move read offset by the length of outer ipv6 + srh
					}
				} 
			}
		}
		
		// WRITE DECAPSULATED PACKETS BACK TO TUN DEVICE
		
		// shrink buffers to avoid sending filler 0's
		for i := range bufs {
			bufs[i] = bufs[i][:r_offset + sizes[i]]
		}

		_, err = worker.TunDev.Write(bufs[:n], w_offset) // offset might have changed if NextHeader == 43
		if err != nil {
			log.Info("SRv6 Proxy: TUN write failed", "interface", worker.IfaceName, "error", err)
		}
	}
}

func (worker *SRv6ProxyWorker) EncapTraffic(ctx context.Context, log logr.Logger){	
	r_offset := VIRTIO_OFFSET
	buffSize := 65535 - r_offset
	buffCount := BUFF_COUNT

	bufs := make([][]byte, buffCount)
	for i := range bufs {
		bufs[i] = make([]byte, r_offset + buffSize)
	}
	sizes := make([]int, buffCount)

	for {
		worker.Cache.Mu.Lock()
		totalSize := worker.Cache.TotalSize
		srv6Header := worker.Cache.Header
		raw := worker.Cache.Raw
		worker.Cache.Mu.Unlock()

		// RESET 
		r_offset := VIRTIO_OFFSET + totalSize //read into buffers with extra space for srv6 header
		w_offset := VIRTIO_OFFSET 

		// resize buffers to original capacity
		for i := range bufs {
			bufs[i] = bufs[i][:cap(bufs[i])]
		}

		n, err := worker.TunDev.Read(bufs, sizes, r_offset) //read into buffers with extra space for srv6 header
		if err != nil {
			log.Info("SRv6 Proxy: TUN read failed", "interface", worker.IfaceName, "error", err)
			return
		}

		for i := range n {
			packet := bufs[i][r_offset : r_offset + sizes[i]]
			if len(packet) == 0 {
				continue
			}

			version := packet[0] >> 4
			
			needsEncap := true
			if version == 6 {
				// filter packets that should not be encapsulated here
				// hlim = 1 packets are not encapped (maybe drop?)
				if packet[7] == 1 {
					needsEncap = false
				}

				// sanity check if cached header version is valid
				if srv6Header.Start[0]>>4 != 6 {
					needsEncap = false
				}
			}

			if needsEncap {
				// copy the cached header into the buffer head room |---Free(virtio)---|------SRv6------|------Packet------|
				copy(bufs[i][w_offset:w_offset+totalSize], raw)
			}
		}

		// shrink buffers to fit the packet sizes
		for i := range bufs {
			bufs[i] = bufs[i][:r_offset + sizes[i]]
		}

		_, err = worker.TunDev.Write(bufs[:n], w_offset) // write whole batch n back
		if err != nil {
			log.Info("SRv6 Proxy: TUN write failed", "interface", worker.IfaceName, "error", err)
			continue
		}
	}
}

func (w *SRv6ProxyWorker) processSRv6Header(packet []byte) (byte, error) {
	start := packet[:24]
	_ = packet[24:40]
	innerNextHeader := packet[40:41]
	hdrExtLen := packet[41:42] // length of segments + options in 8-octets
	routingType := packet[42:43]
	segmentsLeft := packet[43:44] // points to active segment
	lastIndex := packet[44:45] // = FirstSegment
	flags := packet[45:46]
	tag := packet[46:48]

	segmentList := make([][]byte, lastIndex[0] + 1)
	offset := 48
	for i := range segmentList {
		k := offset + i * 16
		segmentList[i] = packet[k:k+16]
	}

	// decrement segmentsLeft (if > 1)
	if segmentsLeft[0] < 1 {
		return 0, fmt.Errorf("Received frame addressed to last segment. Not Caching.")
	}
	activeSID := segmentsLeft[0] - 1

	totalHeaderBytes := hdrExtLen[0] * 8 + byte(48)
	segmentListEndOffset := 48 + len(segmentList) * 16

	updatedHeader := SRv6Header{
		Start: [24]byte(start),
		DstAddr: [16]byte(segmentList[activeSID]), // copy segmentList[segmentsLeft] into dst of outer
		NextHeader: [1]byte(innerNextHeader),
		HdrExtLen: [1]byte(hdrExtLen),
		RoutingType: [1]byte(routingType),
		SegmentsLeft: [1]byte{activeSID}, // is already byte
		FirstSeg: [1]byte(lastIndex),
		Flags: [1]byte(flags),
		Tag: [2]byte(tag),
		SegmentList: bytes.Clone(packet[48:segmentListEndOffset]), // clone explicitly
		Tlvs: bytes.Clone(packet[segmentListEndOffset:totalHeaderBytes]),
	}

	w.Cache.Mu.Lock()
	w.Cache.Header = updatedHeader
	w.Cache.Raw = updatedHeader.ToByte()
	w.Cache.TotalSize = int(totalHeaderBytes)
	w.Cache.Mu.Unlock()
	
	return totalHeaderBytes, nil
}

func (h *SRv6Header) ToByte() []byte {
	bits := append(h.Start[:], h.DstAddr[:]...)
	bits = append(bits, h.NextHeader[0])
	bits = append(bits, h.HdrExtLen[0])
	bits = append(bits, h.RoutingType[0])
	bits = append(bits, h.SegmentsLeft[0])
	bits = append(bits, h.FirstSeg[0])
	bits = append(bits, h.Flags[0])
	bits = append(bits, h.Tag[:]...)
	bits = append(bits, h.SegmentList...)
	bits = append(bits, h.Tlvs...)
	return bits
}
