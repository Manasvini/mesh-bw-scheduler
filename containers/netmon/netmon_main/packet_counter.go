package main

import (
	"encoding/binary"
	"fmt"
	bpf "github.com/iovisor/gobpf/bcc"
	"net"
	"os"
	"sync"
	"time"
)

/*
#cgo CFLAGS: -I/usr/include/bcc/compat
#cgo LDFLAGS: -lbcc
#include <bcc/bcc_common.h>
#include <bcc/libbpf.h>
void perf_reader_free(void *ptr);
*/
import (
	"C"
)

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
	PktStats            *bpf.Table
	PktSize             *bpf.Table
	device              string
	module              *bpf.Module
	lastObservedTraffic map[string][]float64
	lastTs              []int64
	lock                *sync.Mutex
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
	mu := &sync.Mutex{}
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
	bpfRunner := &BPFRunner{lock: mu, PktStats: pktcnt, device: device, module: module, PktSize: pktsize, lastObservedTraffic: make(map[string][]float64, 0)}

	_ = bpfRunner.GetStats()
	_ = bpfRunner.GetStats()
	return bpfRunner
}

func getMean(values []float64, times []int64) float64 {
	sum := 0.0
	if len(values) == 1 {
		return values[0]
	}
	for i := 1; i < len(values); i++ {
		bw := values[i] - values[i-1]
		tdiff := times[i] - times[i-1]
		if tdiff == 0 {
			continue
		}
		sum += (bw / 1000.0) / float64(tdiff)
	}

	return (sum / float64(len(values))) * 1000.0
}
func (runner *BPFRunner) GetStats() map[string]float64 {
	runner.lock.Lock()
	defer runner.lock.Unlock()
	trafficMap := make(map[string][]float64, 0)
	for it := runner.PktSize.Iter(); it.Next(); {
		key := bpf.GetHostByteOrder().Uint32(it.Key())
		value := bpf.GetHostByteOrder().Uint64(it.Leaf())
		trafficMap[fmt.Sprintf("%s", int2ip(key))] = []float64{float64(value)}
		if value > 0 {
			fmt.Printf("%s: %v bytes\n", int2ip(key), value)
		}
	}
	if len(runner.lastObservedTraffic) == 0 {
		runner.lastObservedTraffic = trafficMap
		runner.lastTs = []int64{time.Now().Unix()}
	} else {
		for k, v := range trafficMap {
			runner.lastObservedTraffic[k] = append(runner.lastObservedTraffic[k], v...)
		}
		runner.lastTs = append(runner.lastTs, time.Now().Unix())
	}
	bws := make(map[string]float64, 0)
	for host, vals := range runner.lastObservedTraffic {

		//prevBytes, val := runner.lastObservedTraffic[host]
		meanTraffic := getMean(vals, runner.lastTs)
		bws[host] = 8 * meanTraffic

	}
	//runner.lastObservedTraffic = trafficMap
	return bws
}
func (runner *BPFRunner) PrintStats() {
	runner.lock.Lock()
	defer runner.lock.Unlock()
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
