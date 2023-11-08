package netmon_client

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
		conn, err := grpc.Dial(address, grpc.WithTimeout(300 * time.Second),  grpc.WithTransportCredentials(insecure.NewCredentials()))
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
			if len(path.Hops) > maxPLen {
				maxPLen = len(path.Hops)
			}
		}
	}
	pLen := 2
	logger(fmt.Sprintf("Max p len = %d\n", maxPLen))
	for {
		if pLen > maxPLen {
			break
		}
		for src, pSet := range *paths {
			for dst, path := range pSet {
				curPLen := len(path.Hops)
				if curPLen != pLen {
					continue
				}
				lastHop := path.Hops[curPLen-1]
				curHop := path.Hops[curPLen-2]
				lastBws, lExists := links[lastHop]
				prevPath, pExists := pSet[curHop]
				if lExists && pExists {
					dstBw, dExists := lastBws[lastHop]
					if dExists {
						path.Bandwidth = dstBw.Bandwidth
						if dstBw.Bandwidth > prevPath.Bandwidth {
							path.Bandwidth = prevPath.Bandwidth
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

func (netmonClient *NetmonClient) GetHeadroomStats(ipMap map[string]string, bwReq map[string]map[string]float32) (LinkSet, PathSet, TrafficSet) {
	links := make(LinkSet, 0)
	paths := make(PathSet, 0)
	traffics := make(TrafficSet, 0)
	logger(fmt.Sprintf("Got %d headroom req\n", len(bwReq)))
	for src, reqs := range bwReq {
		for dst, bw := range reqs {
			logger(fmt.Sprintf("src = %s dst  = %s bwreq = %f", src, dst, bw))
		}
	}
	for addr, _ := range netmonClient.clients {
		host := strings.Split(addr, ":")[0]

		logger(fmt.Sprintf("addr = %s", host))	
		curLinks, curPaths, curTraffic := netmonClient.getStatsOneHeadroom(addr, ipMap, bwReq[host])
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
				logger(fmt.Sprintf("added path %s to %s with bw = %f and %d hops\n", host, dst, path.Bandwidth, len(path.Hops)))
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


func (netmonClient *NetmonClient) GetStats(ipMap map[string]string) (LinkSet, PathSet, TrafficSet) {
	links := make(LinkSet, 0)
	paths := make(PathSet, 0)
	traffics := make(TrafficSet, 0)
	for addr, _ := range netmonClient.clients {
		curLinks, curPaths, curTraffic := netmonClient.getStatsOne(addr, ipMap)
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
				logger(fmt.Sprintf("added path %s to %s with bw = %f and %d hops\n", host, dst, path.Bandwidth, len(path.Hops)))
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

func (netmonClient *NetmonClient) getStatsOne(address string, ipMap map[string]string) (LinkSet, PathSet, TrafficSet) {
	fmt.Printf("host= " + address)
	host := strings.Split(address, ":")[0]
	logger(fmt.Sprintf("address = %s", host))
	client, exists := netmonClient.clients[address]
	if !exists {
		var links LinkSet
		var paths PathSet
		traffic := make(TrafficSet, 0)
		return links, paths, traffic
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	response, err := client.GetNetInfo(ctx, &pb.NetInfoRequest{})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	return netmonClient.ProcessResponse(response, host, ipMap)
}

func (netmonClient *NetmonClient) getStatsOneHeadroom(address string, ipMap map[string]string, bwReq map[string]float32) (LinkSet, PathSet, TrafficSet) {
	fmt.Printf("host= " + address)
	host := strings.Split(address, ":")[0]
	logger(fmt.Sprintf("address = %s got %d req", host, len(bwReq)))
	client, exists := netmonClient.clients[address]
	if !exists {
		var links LinkSet
		var paths PathSet
		traffic := make(TrafficSet, 0)
		return links, paths, traffic
	}
	bwInfos := make([]*pb.BandwidthInfo, 0)
	for h, bw := range bwReq {
		logger(fmt.Sprintf("added %s with req %f", h, bw))
		bwInfos = append(bwInfos, &pb.BandwidthInfo{Host:h, SendBw: bw})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15 * time.Second)
	defer cancel()
	response, err := client.GetHeadroomInfo(ctx, &pb.HeadroomInfoRequest{BwInfo:bwInfos})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	return netmonClient.ProcessResponse(response, host, ipMap)
}

func (netmonClient *NetmonClient) ProcessResponse(response *pb.NetInfoReply, host string, ipMap map[string]string) (LinkSet, PathSet, TrafficSet) {
	var links LinkSet
	var paths PathSet
	traffic := make(TrafficSet, 0)
	bwInfos := response.BwInfo
	
	logger(fmt.Sprintf("Got %d bw stats\n", len(bwInfos)))
	trInfo := response.TrInfo
	logger(fmt.Sprintf("Got %d traceroutes\n", len(trInfo)))

	paths = make(PathSet, 0)
	pMap := make(map[string]Path, 0)
	paths[host] = pMap
	for _, tr := range trInfo {
		for _, hop := range tr.Hops {
			logger(fmt.Sprintf("src = %s dst = %s hop = %s\n", host, tr.Host, hop))
		}
		path := Path{Source: host, Destination: tr.Host, Hops: tr.Hops}
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
		link := Link{Source: host, Destination: bw.Host, Bandwidth: float64(bw.SendBw)}
		_, exists := lMap[bw.Host]
		if !exists {
			lMap[bw.Host] = link
		}

		tSrc := bw.Host
		tDst := host
		tMap, exists := traffic[tSrc]
		if !exists {
			tMap = make(map[string]Traffic, 0)
		}

		tMap[tDst] = Traffic{Source: tSrc, Destination: tDst, Bytes: float64(bw.RecvBwUsed)}
		logger(fmt.Sprintf("Got traffic src = %s dst = %s bytes = %f\n", tSrc, tDst, bw.RecvBwUsed))
		traffic[tSrc] = tMap

		path, exists := pMap[bw.Host]
		logger(fmt.Sprintf("src = %s dst = %s hops=%d bw = %f\n", host, bw.Host, len(path.Hops), bw.SendBw))
		pathActual := make([]string, 0)
		for _, p := range path.Hops {
			logger(fmt.Sprintf("hop = %s actual = %s\n", p, ipMap[p]))
			if strings.Contains(p, "*") {
				continue
			}
			ipActual, exists := ipMap[p]
			if !exists {
				ipActual = p
			}
			if len(pathActual) > 0 && ipActual != bw.Host{
				pathActual = append(pathActual, ipActual)
			} else if len(pathActual) == 0{
				pathActual = append(pathActual, ipActual)
			}
		}
		path.Hops = pathActual
		logger(fmt.Sprintf("no. of hops = %d\n", len(path.Hops)))
		if exists && len(path.Hops) == 1 {
			path.Bandwidth = float64(bw.SendBw)
		}
		pMap[bw.Host] = path
		logger(fmt.Sprintf("Got bw for %s to %s = %f\n", host, bw.Host, path.Bandwidth))
	}
	links[host] = lMap
	paths[host] = pMap
	return links, paths, traffic
}
