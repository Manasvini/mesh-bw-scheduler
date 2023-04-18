package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type NetmonClient struct {
	clients     map[string]pb.NetMonitorClient
	clientConns map[string]*grpc.ClientConn
}

func NewNetmonClient(addresses []string) *NetmonClient {
	conns := make(map[string]*grpc.ClientConn, 0)
	clients := make(map[string]pb.NetMonitorClient, 0)
	for _, address := range addresses {
		conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		client := pb.NewNetMonitorClient(conn)
		clients[address] = client
		conns[address] = conn
	}
	return &NetmonClient{clients: clients, clientConns: conns}

}

func (netmonClient *NetmonClient) Close() {
	for _, conn := range netmonClient.clientConns {
		conn.Close()
	}
}

func (netmonClient *NetmonClient) ComputePathBw(links LinkSet, paths *PathSet) {
	maxPLen := 1
	for _, pSet := range *paths {
		for _, path := range pSet {
			if len(path.hops) > maxPLen {
				maxPLen = len(path.hops)
			}
		}
	}
	pLen := 2
	fmt.Printf("Max p len = %d\n", maxPLen)
	for {
		if pLen > maxPLen {
			break
		}
		for src, pSet := range *paths {
			for dst, path := range pSet {
				curPLen := len(path.hops)
				if curPLen != pLen {
					continue
				}
				lastHop := path.hops[curPLen-1]
				curHop := path.hops[curPLen-2]
				lastBws, lExists := links[lastHop]
				prevPath, pExists := pSet[curHop]
				if lExists && pExists {
					dstBw, dExists := lastBws[lastHop]
					if dExists {
						path.bandwidth = dstBw.bandwidth
						if dstBw.bandwidth > prevPath.bandwidth {
							path.bandwidth = prevPath.bandwidth
						}
						pSet[dst] = path
					}
				}
			}
			(*paths)[src] = pSet
		}
		pLen += 1

	}
}

func (netmonClient *NetmonClient) GetStats() {
	links := make(LinkSet, 0)
	paths := make(PathSet, 0)
	for addr, _ := range netmonClient.clients {
		curLinks, curPaths := netmonClient.GetStatsOne(addr)
		if curLinks != nil && curPaths != nil {
			_, existsP := curPaths[addr]
			_, existsL := curLinks[addr]
			if !existsP || !existsL {
				fmt.Printf("No links or paths\n")
				continue
			}
			paths[addr] = curPaths[addr]
			links[addr] = curLinks[addr]
			fmt.Printf("add %d links to %s \n", len(curLinks[addr]), addr)
		}
	}
	netmonClient.ComputePathBw(links, &paths)
}

func (netmonClient *NetmonClient) GetStatsOne(address string) (LinkSet, PathSet) {
	var links LinkSet
	var paths PathSet

	client, exists := netmonClient.clients[address]
	if !exists {
		return links, paths
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	response, err := client.GetNetInfo(ctx, &pb.NetInfoRequest{})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	bwInfos := response.BwInfo
	fmt.Printf("Got %d bw stats\n", len(bwInfos))
	trInfo := response.TrInfo
	fmt.Printf("Got %d traceroutes\n", len(trInfo))

	paths = make(PathSet, 0)
	pMap := make(map[string]Path, 0)
	paths[address] = pMap
	for _, tr := range trInfo {
		path := Path{source: address, destination: tr.Host, hops: tr.Hops}
		_, exists := pMap[tr.Host]
		if !exists {
			pMap[tr.Host] = path
		}
	}
	paths[address] = pMap

	links = make(LinkSet, 0)
	lMap := make(map[string]Link, 0)
	links[address] = lMap
	for _, bw := range bwInfos {
		link := Link{source: address, destination: bw.Host, bandwidth: float64(bw.SendBw)}
		_, exists = lMap[bw.Host]
		if !exists {
			lMap[bw.Host] = link
		}
		path, exists := pMap[bw.Host]
		if exists && len(path.hops) == 1 && address != bw.Host {
			path.bandwidth = float64(bw.SendBw)
		}
	}
	links[address] = lMap
	return links, paths
}
