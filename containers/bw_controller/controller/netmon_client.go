package bw_controller

import (
	"context"
	"fmt"
	"log"
	"strings"
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

func (netmonClient *NetmonClient) GetStats() (LinkSet, PathSet, TrafficSet) {
	links := make(LinkSet, 0)
	paths := make(PathSet, 0)
	traffics := make(TrafficSet, 0)
	for addr, _ := range netmonClient.clients {
		curLinks, curPaths, curTraffic := netmonClient.GetStatsOne(addr)
		if curLinks != nil && curPaths != nil {
			host := strings.Split(addr, ":")[0]
			srcPaths, existsP := curPaths[host]
			srcLinks, existsL := curLinks[host]
			if !existsP || !existsL {
				fmt.Printf("No links or paths\n")
				continue
			}
			_, exists := links[host]
			if !exists {
				links[host] = make(map[string]Link, 0)
			}
			_, exists = paths[host]
			if !exists {
				paths[host] = make(map[string]Path, 0)
			}

			for src, tSrcInfo := range curTraffic {
				_, exists := traffics[src]
				if !exists {
					traffics[src] = make(map[string]Traffic, 0)
				}
				for dst, traffic := range tSrcInfo {
					traffics[src][dst] = traffic
				}
			}
			for dst, path := range srcPaths {
				paths[host][dst] = path
				logger(fmt.Sprintf("added path %s to %s with bw = %f\n", host, dst, path.bandwidth))
			}
			for dst, link := range srcLinks {
				links[host][dst] = link
			}

			fmt.Printf("add %d links to %s \n", len(curLinks[host]), host)
		}
	}
	netmonClient.ComputePathBw(links, &paths)
	return links, paths, traffics
}

func (netmonClient *NetmonClient) GetStatsOne(address string) (LinkSet, PathSet, TrafficSet) {
	var links LinkSet
	var paths PathSet
	logger("host= " + address)
	traffic := make(TrafficSet, 0)

	host := strings.Split(address, ":")[0]
	client, exists := netmonClient.clients[address]
	if !exists {
		return links, paths, traffic
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	response, err := client.GetNetInfo(ctx, &pb.NetInfoRequest{})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	bwInfos := response.BwInfo
	logger(fmt.Sprintf("Got %d bw stats\n", len(bwInfos)))
	trInfo := response.TrInfo
	logger(fmt.Sprintf("Got %d traceroutes\n", len(trInfo)))

	paths = make(PathSet, 0)
	pMap := make(map[string]Path, 0)
	paths[host] = pMap
	for _, tr := range trInfo {
		path := Path{source: host, destination: tr.Host, hops: tr.Hops}
		_, exists := pMap[tr.Host]
		if !exists {
			pMap[tr.Host] = path
		}
	}
	paths[host] = pMap

	links = make(LinkSet, 0)
	lMap := make(map[string]Link, 0)

	links[host] = lMap
	for _, bw := range bwInfos {
		link := Link{source: host, destination: bw.Host, bandwidth: float64(bw.SendBw)}
		_, exists = lMap[bw.Host]
		if !exists {
			lMap[bw.Host] = link
		}

		tSrc := bw.Host
		tDst := host
		tMap, exists := traffic[tSrc]
		if !exists {
			tMap = make(map[string]Traffic, 0)
		}

		tMap[tDst] = Traffic{source: tSrc, destination: tDst, bytes: float64(bw.RecvBwUsed)}
		logger(fmt.Sprintf("Got traffic src = %s dst = %s bytes = %f\n", tSrc, tDst, bw.RecvBwUsed))
		traffic[tSrc] = tMap

		path, exists := pMap[bw.Host]
		logger(fmt.Sprintf("src = %s dst = %s hops=%d bw = %f\n", host, bw.Host, len(path.hops), bw.SendBw))
		if exists && len(path.hops) == 1 && host != bw.Host {
			path.bandwidth = float64(bw.SendBw)
		}
		pMap[bw.Host] = path
		logger(fmt.Sprintf("Got bw for %s to %s = %f\n", host, bw.Host, bw.SendBw))
	}
	links[host] = lMap
	return links, paths, traffic
}
