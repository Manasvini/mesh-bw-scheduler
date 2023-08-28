package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"time"
	bpf "github.com/iovisor/gobpf/bcc"
)

/*
#cgo CFLAGS: -I/usr/include/bcc/compat
#cgo LDFLAGS: -lbcc
#include <bcc/bcc_common.h>
#include <bcc/libbpf.h>
void perf_reader_free(void *ptr);
*/
import "C"

const source string = `
#include "packet.h"
BPF_HASH(packets);
BPF_HASH(packetsize);

int hello_packet(struct xdp_md *ctx) {
    u64 counter = 0;
	u64 cur_pkt_size=0;
	u64 size = 0;
	u64 *pktsize;
    u64 key = 0;
    u64 *p;
    key = parse_ipv4_dest(ctx);
	cur_pkt_size = get_pkt_size(ctx);
    if (key != 0) {
        p = packets.lookup(&key);
        if (p != 0) {
            counter = *p;
        }
        counter++;
        packets.update(&key, &counter);
	}
	if (key != 0){
		p = packetsize.lookup(&key);
		if (p != 0) {
			size = *p;
		}
		size +=  cur_pkt_size;
		packetsize.update(&key, &size);
    }

    return XDP_PASS;
}
`

func usage() {
	fmt.Printf("Usage: %v <ifdev>\n", os.Args[0])
	fmt.Printf("e.g.: %v eth0\n", os.Args[0])
	os.Exit(1)
}
func int2ip(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.LittleEndian.PutUint32(ip, nn)
	return ip
}

type BPFRunner struct {
	PktStats *bpf.Table
	PktSize  *bpf.Table
	device   string
	module   *bpf.Module
	lastObservedTraffic map[string]float64
	lastTs 	int64
}

func (runner *BPFRunner) Close() {
	defer runner.module.Close()
	defer func() {
		if err := runner.module.RemoveXDP(runner.device); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to remove XDP from %s: %v\n", device, err)
		}
	}()
}

func NewBPFRunner(device string) *BPFRunner {
	module := bpf.NewModule(source, []string{})

	fn, err := module.Load("hello_packet", C.BPF_PROG_TYPE_XDP, 1, 65536)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load xdp prog: %v\n", err)
		os.Exit(1)
	}

	err = module.AttachXDPWithFlags(device, fn, bpf.XDP_FLAGS_SKB_MODE)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to attach xdp prog: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Counting packets, hit CTRL+C to stop")

	pktcnt := bpf.NewTable(module.TableId("packets"), module)
	pktsize := bpf.NewTable(module.TableId("packetsize"), module)
	bpfRunner := &BPFRunner{PktStats: pktcnt, device: device, module: module, PktSize: pktsize, lastObservedTraffic:make(map[string]float64, 0)}

	return bpfRunner
}

func (runner *BPFRunner) GetStats() map[string]float64 {
	trafficMap := make(map[string]float64, 0)
	for it := runner.PktSize.Iter(); it.Next(); {
		key := bpf.GetHostByteOrder().Uint32(it.Key())
		value := bpf.GetHostByteOrder().Uint64(it.Leaf())
		trafficMap[fmt.Sprintf("%s", int2ip(key))] = float64(value)
		if value > 0 {
			fmt.Printf("%s: %v bytes\n", int2ip(key), value)
		}
	}
	if len(runner.lastObservedTraffic) == 0 {
		runner.lastObservedTraffic = trafficMap
		runner.lastTs = time.Now().Unix()
		return trafficMap
	}
	bws := make(map[string]float64, 0)
	for host, bytes := range trafficMap {
		prevBytes, val := runner.lastObservedTraffic[host]
		if val {
			bws[host] = (bytes - prevBytes)/ float64(time.Now().Unix() - runner.lastTs)
		}
	}
	runner.lastTs = time.Now().Unix()
	runner.lastObservedTraffic = trafficMap
	return bws
}
func (runner *BPFRunner) PrintStats() {
	fmt.Printf("\n{IP address}: {total pkts}\n")
	for it := runner.PktStats.Iter(); it.Next(); {
		key := bpf.GetHostByteOrder().Uint32(it.Key())
		value := bpf.GetHostByteOrder().Uint64(it.Leaf())

		if value > 0 {
			fmt.Printf("%s: %v pkts\n", int2ip(key), value)
		}
	}
	fmt.Printf("\n{IP address}: {total bytes}\n")
	for it := runner.PktSize.Iter(); it.Next(); {
		key := bpf.GetHostByteOrder().Uint32(it.Key())
		value := bpf.GetHostByteOrder().Uint64(it.Leaf())

		if value > 0 {
			fmt.Printf("%s: %v bytes\n", int2ip(key), value)
		}
	}
}
