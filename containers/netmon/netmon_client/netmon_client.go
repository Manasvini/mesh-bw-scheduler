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

func (netmonClient *NetmonClient) ComputePathTraffic(traffic TrafficSet, pathsInput PathSet)TrafficSet {
	for _, pSet := range pathsInput {
		for dst, path := range pSet {
			if len(path.Hops) <= 1{
				continue
			}
			//logger(fmt.Sprintf("src = %s dst = %s hop0 = %s\n", src, dst, path.Hops[0]))
			cumulative_traffic := 0.0
			cursrc := path.Hops[0]
			curdst := path.Hops[1]
			bytes_to_add := 0.0
			exists := false
			if _, exists = traffic[cursrc]; exists {
				_, exists = traffic[cursrc][curdst]
			}
			if exists{
				bytes_to_add = traffic[cursrc][curdst].Bytes
				cumulative_traffic = bytes_to_add
			}
			for i := 1; i < len(path.Hops)- 1; i++{
				cursrc := path.Hops[i]
				curdst := path.Hops[i+1]
				exists := false
				if _, exists = traffic[cursrc]; exists {
					_, exists = traffic[cursrc][curdst]
				}
				if exists{
					bytes_to_add = traffic[cursrc][curdst].Bytes
				}
				cumulative_traffic += bytes_to_add
				//logger(fmt.Sprintf("i=%d hop=  %s traffic=%f cumul=%f\n", i, path.Hops[i], bytes_to_add, cumulative_traffic))
				if exists{
					traffic[cursrc][curdst] = Traffic{Source:cursrc, Destination:curdst, Bytes:cumulative_traffic}
				}
			}
			cursrc = path.Hops[len(path.Hops)-1]
			curdst = dst
			if _, exists = traffic[cursrc]; exists {
				_, exists = traffic[cursrc][curdst]
			}
			if exists{
				bytes_to_add = traffic[cursrc][curdst].Bytes
				cumulative_traffic += bytes_to_add
				traffic[cursrc][curdst] = Traffic{Source:cursrc, Destination:curdst, Bytes:cumulative_traffic}
			
			}

			//logger(fmt.Sprintf("src=%s dst=%s traffic=%f cumul=%f\n", src, dst, bytes_to_add, traffic[cursrc][curdst]))
			
		}
	}
	return traffic
}

func (netmonClient *NetmonClient) ComputePathBw(links LinkSet, pathsInput PathSet) PathSet {
	maxPLen := 1
	paths := make(PathSet, 0)
	for _, pSet := range pathsInput {
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
		for src, pSetIn := range pathsInput {
			pSet := make(map[string]Path, 0)
			paths[src] = pSet
			for dst, path := range pSetIn {
				pSet[dst] = path
			}
			for dst, path := range pSet {
				curPLen := len(path.Hops)
				if curPLen != pLen {
					continue
				}
				lastHop := path.Hops[curPLen-1]
				curHop := path.Hops[curPLen-2]
				logger(fmt.Sprintf("src = %s dst=%s lasthop %s curHop %s cur plen = %d pLen = %d", src, dst, lastHop, curHop, curPLen, pLen))
				lastBws, lExists := links[lastHop]
				if curPLen == 2{
					curHop = lastHop
				}
				prevPath, pExists := pSet[curHop]
				if lExists && pExists {
					dstBw, dExists := lastBws[dst]
					if dExists {
						logger(fmt.Sprintf("Path exists via routing link bw = %f path bw = %f", dstBw.Bandwidth, prevPath.Bandwidth))
						path.Bandwidth = dstBw.Bandwidth
						if dstBw.Bandwidth > prevPath.Bandwidth {
							path.Bandwidth = prevPath.Bandwidth
						}
						pSet[dst] = path
						logger(fmt.Sprintf("path bw = %f", pSet[dst].Bandwidth))
					}
				}
			}
			paths[src] = pSet
		}
		pLen += 1

	}
	return paths
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
				for dst, traffic := range tSrcInfo {
					_, exists := traffics[dst]
					if !exists {
						traffics[dst] = make(map[string]Traffic, 0)
					}
					traffics[dst][src] = traffic
				}
			}
			for dst, path := range srcPaths {
				paths[host][dst] = path
				logger(fmt.Sprintf("added path %s to %s with bw = %f and %d hops\n", host, dst, path.Bandwidth, len(path.Hops)))
			}
			for dst, link := range srcLinks {
				links[host][dst] = link
			}

			//fmt.Printf("add %d links to %s \n", len(curLinks[host]), host)
		}
	}
	pathsOut  := netmonClient.ComputePathBw(links, paths)
	traffics = netmonClient.ComputePathTraffic(traffics, paths)
	for src, dstPaths := range pathsOut {
		for dst, path := range dstPaths {
			logger(fmt.Sprintf("src = %s dst = %s bw = %f\n", src, dst, path.Bandwidth))
		} 
	}
	return links, pathsOut, traffics
}


func (netmonClient *NetmonClient) GetStats(ipMap map[string]string, bwUpdate bool) (LinkSet, PathSet, TrafficSet) {
	links := make(LinkSet, 0)
	paths := make(PathSet, 0)
	traffics := make(TrafficSet, 0)
	for addr, _ := range netmonClient.clients {
		curLinks, curPaths, curTraffic := netmonClient.getStatsOne(addr, ipMap, bwUpdate)
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
				for dst, traffic := range tSrcInfo {
					_, exists := traffics[dst]
					if !exists {
						traffics[dst] = make(map[string]Traffic, 0)
					}
					traffics[dst][src] = traffic
				}
			}
			for dst, path := range srcPaths {
				paths[host][dst] = path
				//logger(fmt.Sprintf("added path %s to %s with bw = %f and %d hops\n", host, dst, path.Bandwidth, len(path.Hops)))
			}
			for dst, link := range srcLinks {
				links[host][dst] = link
			}

			//fmt.Printf("add %d links to %s \n", len(curLinks[host]), host)
		}
	}
	pathsOut := netmonClient.ComputePathBw(links, paths)
	traffics = netmonClient.ComputePathTraffic(traffics, paths)
	
	//for src, trafs := range traffics {
	//	for dst, traf := range trafs {
	//		logger(fmt.Sprintf("src = %s dst = %s bw = %f\n", src, dst, traf.Bytes))
	//	} 
	//}
	//for src, dstPaths := range pathsOut {
	//	for dst, path := range dstPaths {
	//		logger(fmt.Sprintf("src = %s dst = %s bw = %f\n", src, dst, path.Bandwidth))
	//	} 
	//}
	return links, pathsOut, traffics
}

func (netmonClient *NetmonClient) getStatsOne(address string, ipMap map[string]string, bwUpdate bool) (LinkSet, PathSet, TrafficSet) {
	//fmt.Printf("host= " + address)
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
	response, err := client.GetNetInfo(ctx, &pb.NetInfoRequest{ShouldUpdate:bwUpdate})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	return netmonClient.ProcessResponse(response, host, ipMap)
}

func (netmonClient *NetmonClient) getStatsOneHeadroom(address string, ipMap map[string]string, bwReq map[string]float32) (LinkSet, PathSet, TrafficSet) {
	//fmt.Printf("host= " + address)
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
	
	//logger(fmt.Sprintf("Got %d bw stats\n", len(bwInfos)))
	trInfo := response.TrInfo
	//logger(fmt.Sprintf("Got %d traceroutes\n", len(trInfo)))

	paths = make(PathSet, 0)
	pMap := make(map[string]Path, 0)
	paths[host] = pMap

	for _, tr := range trInfo {
		pathActual := make([]string, 0)
	//	for _, hop := range tr.Hops {
	//		logger(fmt.Sprintf("src = %s dst = %s hop = %s\n", host, tr.Host, hop))
	//	}
		path := Path{Source: host, Destination: tr.Host, Hops: tr.Hops}
		for _, p := range path.Hops {
			//logger(fmt.Sprintf("src = %s dst = %s hop = %s actual = %s\n", path.Source, path.Destination, p, ipMap[p]))
			//for _, h := range pathActual {
			//	logger(fmt.Sprintf("%s ", h))
			//}
			if len(pathActual) < 1{
				pathActual = append(pathActual, host)
			}
			if strings.Contains(p, "*"){
				continue
			}
			ipActual, exists := ipMap[p]
			if !exists {
				continue
				//ipActual = p
			}
			if len(pathActual) >= 1 && ipActual != tr.Host && ipActual != host {
				pathActual = append(pathActual, ipActual)
			} 
			if ipActual == tr.Host {
				break
			}
		}
		path.Hops = pathActual
		//logger(fmt.Sprintf("actual plen = %d", len(path.Hops)))
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
		link := Link{Source: host, Destination: bw.Host, Bandwidth: float64(bw.ReceiveBw)}
		_, exists := lMap[bw.Host]
		if !exists {
			lMap[bw.Host] = link
		}
		//logger(fmt.Sprintf("Got bw avail src = %s dst = %s bw = %f\n", host, bw.Host, bw.SendBw))
		tSrc := bw.Host
		tDst := host
		tMap, exists := traffic[tSrc]
		if !exists {
			tMap = make(map[string]Traffic, 0)
		}

		tMap[tDst] = Traffic{Source: tSrc, Destination: tDst, Bytes: float64(bw.RecvBwUsed)}
		//logger(fmt.Sprintf("Got traffic src = %s dst = %s bytes = %f\n", tSrc, tDst, bw.RecvBwUsed))
		traffic[tSrc] = tMap

		path, exists := pMap[bw.Host]
		//logger(fmt.Sprintf("src = %s dst = %s hops=%d bw = %f exists = %v\n", host, bw.Host, len(path.Hops), bw.SendBw, exists))
		/*pathActual := make([]string, 0)
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
		logger(fmt.Sprintf("no. of hops = %d\n", len(path.Hops)))*/
		if exists && len(path.Hops) <= 1 {
			path.Bandwidth = float64(bw.SendBw)
		}
		pMap[bw.Host] = path
		//logger(fmt.Sprintf("Got bw for %s to %s = %f\n", host, bw.Host, path.Bandwidth))
	}
	links[host] = lMap
	paths[host] = pMap
	return links, paths, traffic
}
