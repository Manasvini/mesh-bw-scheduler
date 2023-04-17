package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"

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
int hello_packet(struct xdp_md *ctx) {
    u64 counter = 0;
    u64 key = 0;
    u64 *p;
    key = parse_ipv4_dest(ctx);
    if (key != 0) {
        p = packets.lookup(&key);
        if (p != 0) {
            counter = *p;
        }
        counter++;
        packets.update(&key, &counter);
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
	device   string
	module   *bpf.Module
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

	err = module.AttachXDP(device, fn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to attach xdp prog: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Counting packets, hit CTRL+C to stop")

	pktcnt := bpf.NewTable(module.TableId("packets"), module)
	bpfRunner := &BPFRunner{PktStats: pktcnt, device: device, module: module}

	return bpfRunner
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
}
