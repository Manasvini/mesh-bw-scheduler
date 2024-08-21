package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	pb "github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon"
	"google.golang.org/grpc"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

var (
	device = flag.String("device", "ens2", "device to monitor")
)

var (
	config = flag.String("config", "config.txt", "config file with list of hosts")
)
var (
	helper = flag.String("helper", "0.0.0.0:6000", "net helper ip/port")
)

func readConfig(configfile string) []string {
	body, err := ioutil.ReadFile(configfile)
	if err != nil {
		log.Fatalf("unable to read file: %v", err)
	}
	hosts := strings.Split(string(body), "\n")
	hostsFinal := make([]string, 0)
	for _, h := range hosts {
		if len(h) > 1 {
			hostsFinal = append(hostsFinal, h)
		}
		fmt.Printf("Got host %s\n", h)
	}
	return hostsFinal
}

type server struct {
	pb.UnimplementedNetMonitorServer
	BwCache                map[string]Bandwidth // dest node -> bw map [ to estimate link capacity]
	HeadroomCacheMeasured  map[string]Bandwidth // dest node -> available bw [to check if excess capacity is available. The goal is to avoid disrupting existing flows. The headroom bw is specified by the controller]
	HeadroomCacheRequested map[string]pb.BandwidthInfo
	TrCache                TracerouteResults // dest node -> traceroute map
	LatencyCache           LatencyResults    // dest node -> latency map
	mu                     sync.Mutex
	netClient              http.Client
	hosts                  []string
	bpfRunner              *BPFRunner
	hostIdx                int
	pendingBwRequest       bool
	headroomIdx            int
}

func (s *server) QueryNetStats(hostname string, qty string) ([]byte, error) {
	reqURL := "http://" + *helper
	fmt.Printf("host = %s\n", hostname)
	res, err := s.netClient.Get(reqURL + "/" + qty + "?host=" + hostname)
	if err != nil {
		fmt.Printf("client: could not create request: %s\n", err)
		return []byte(""), err
	}

	fmt.Printf("client: status code: %d\n", res.StatusCode)

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("client: could not read response body: %s\n", err)
		return []byte(""), err
	}
	fmt.Printf(string(resBody))
	return resBody, nil

}

func (s *server) QueryHeadroom(hostname string, bw float32) ([]byte, error) {
	reqURL := "http://" + *helper
	fmt.Printf("host = %s bw = %f\n", hostname, bw)
	res, err := s.netClient.Get(reqURL + "/bw" + "?host=" + hostname + "&bwmax=" + fmt.Sprintf("%f", bw))
	if err != nil {
		fmt.Printf("client: could not create request: %s\n", err)
		return []byte(""), err
	}

	fmt.Printf("client: status code: %d\n", res.StatusCode)

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("client: could not read response body: %s\n", err)
		return []byte(""), err
	}
	fmt.Printf(string(resBody))
	return resBody, nil

}
func (s *server) QueryTrStats() ([]byte, error) {
	reqURL := "http://" + *helper
	res, err := s.netClient.Get(reqURL + "/traceroute")
	if err != nil {
		fmt.Printf("client: could not create request: %s\n", err)
		return []byte(""), err
	}

	fmt.Printf("client: status code: %d\n", res.StatusCode)

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("client: could not read response body: %s\n", err)
		return []byte(""), err
	}
	fmt.Printf(string(resBody))
	return resBody, nil
}

func GetBwResults(bwResponse []byte) BandwidthResults {
	var bwInfo BandwidthResults
	json.Unmarshal(bwResponse, &bwInfo)
	fmt.Printf("\noriginal %s unmarshal %v\n", string(bwResponse), bwInfo)
	return bwInfo
}

func GetTrResults(trResponse []byte) TracerouteResults {
	var trInfo TracerouteResults
	json.Unmarshal(trResponse, &trInfo)
	return trInfo
}

func GetLatencyResults(latencyResponse []byte) LatencyResults {
	var latencyInfo LatencyResults
	json.Unmarshal(latencyResponse, &latencyInfo)
	return latencyInfo
}

func (s *server) GetNetInfo(ctx context.Context, in *pb.NetInfoRequest) (*pb.NetInfoReply, error) {
	log.Printf("Received: req")
	//s.mu.Lock()
	bwUsed := s.bpfRunner.GetStats()
	bwInfos := make([]*pb.BandwidthInfo, 0)
	bws := s.BwCache
	trs := s.TrCache
	//s.mu.Unlock()
	if !s.pendingBwRequest {
		s.pendingBwRequest = in.ShouldUpdate
	}
	for dst, trafficSent := range bwUsed {
		bw, exists := bws[dst]
		log.Printf("BWInfoFull: Host = %s bw = %f", bw.Host, bw.RcvBw)
		bwInfo := pb.BandwidthInfo{Host: dst, RecvBwUsed: float32(trafficSent)}
		if exists {
			bwInfo.SendBw = float32(bw.SndBw)
			bwInfo.ReceiveBw = float32(bw.RcvBw)
		}
		bwInfos = append(bwInfos, &bwInfo)
	}
	log.Printf("Got %d bws", len(bwInfos))
	trInfos := make([]*pb.TracerouteInfo, 0)
	for _, tr := range trs.TracerouteResults {
		trInfo := pb.TracerouteInfo{Host: tr.Host, Hops: tr.Route}
		trInfos = append(trInfos, &trInfo)
	}
	reply := &pb.NetInfoReply{BwInfo: bwInfos, TrInfo: trInfos}
	return reply, nil
}

func (s *server) GetHeadroomInfo(ctx context.Context, in *pb.HeadroomInfoRequest) (*pb.NetInfoReply, error) {
	//s.mu.Lock()
	//defer s.mu.Unlock()
	bwUsed := s.bpfRunner.GetStats()
	hrInfos := make([]*pb.BandwidthInfo, 0)
	log.Printf("cahce has %d measure and  %d reqs , req has %d bws", len(s.HeadroomCacheMeasured), len(s.HeadroomCacheRequested), len(in.BwInfo))
	for dst, trafficSent := range bwUsed {
		//requestedHeadroomInfo, exists  := s.HeadroomCacheRequested[hostInfo.Host]
		var hostInfo *pb.BandwidthInfo = nil
		for _, h := range in.BwInfo {
			if h.Host == dst {
				hostInfo = h
			}
		}
		log.Printf("Recv bw Used = %f\n", trafficSent)
		bwInfo := pb.BandwidthInfo{RecvBwUsed: float32(trafficSent)}
		bwInfo.Host = dst
		if hostInfo != nil {
			log.Printf("host = %s headroom req = %f %v", bwInfo.Host, hostInfo.SendBw, hostInfo)
			s.HeadroomCacheRequested[hostInfo.Host] = *hostInfo

			measuredHeadroom, exists := s.HeadroomCacheMeasured[hostInfo.Host]
			if exists {
				bwInfo.ReceiveBw = float32(measuredHeadroom.RcvBw)
				bwInfo.SendBw = float32(measuredHeadroom.SndBw)
			}
		}
		hrInfos = append(hrInfos, &bwInfo)
		//if exists {
		//	headroomInfo := pb.BandwidthInfo{Host:measuredHeadroom.Host, SendBw: float32(measuredHeadroom.SndBw), ReceiveBw: float32(measuredHeadroom.RcvBw)}
		//	trafficSent, exists := bwUsed[hostInfo.Host]
		//	if exists {
		//		headroomInfo.RecvBwUsed = float32(trafficSent)
		//	}
		//	hrInfos = append(hrInfos, &headroomInfo)
		//}

	}
	trInfos := make([]*pb.TracerouteInfo, 0)
	for _, tr := range s.TrCache.TracerouteResults {
		trInfo := pb.TracerouteInfo{Host: tr.Host, Hops: tr.Route}
		trInfos = append(trInfos, &trInfo)
	}

	reply := &pb.NetInfoReply{BwInfo: hrInfos, TrInfo: trInfos}
	return reply, nil
}

func (s *server) GetUpdatedNetStats() (BandwidthResults, TracerouteResults, LatencyResults) {
	var allBwInfo BandwidthResults
	var allLatencyInfo LatencyResults
	var trInfo TracerouteResults
	//for _, host := range s.hosts {
		s.pendingBwRequest = true
		host := s.hosts[s.hostIdx]
		fmt.Printf("host = %s idx = %d\n", host, s.hostIdx)
		bwResponse, err := s.QueryNetStats(host, "bw")
		if err == nil {
			bwInfo := GetBwResults(bwResponse)
			if len(bwInfo.BandwidthResults) > 0 {

				log.Printf("Update stat for %s rcv bw = %f snd bw = %f %v", host, bwInfo.BandwidthResults[0].RcvBw, bwInfo.BandwidthResults[0].SndBw, bwInfo)
				allBwInfo.BandwidthResults = append(allBwInfo.BandwidthResults, bwInfo.BandwidthResults[0])
				s.hostIdx = (s.hostIdx + 1)

			}
		}
	//}
	trResponse, err := s.QueryTrStats()
	if err == nil {
		trInfo = GetTrResults(trResponse)
	}
	fmt.Printf("tr: Got %d hosts\n", len(trInfo.TracerouteResults))

	fmt.Printf("bw: Got %d hosts\n", len(allBwInfo.BandwidthResults))
	return allBwInfo, trInfo, allLatencyInfo
}

func (s *server) GetUpdatedHeadroomStats() (BandwidthResults, TracerouteResults) {
	var headroomInfo BandwidthResults
	var trInfo TracerouteResults
	host := s.hosts[s.headroomIdx]
	//for _, host := range s.hosts {
	fmt.Printf("headroom host = %s headroom idx = %d\n", host, s.headroomIdx)
	headroomReq, exists := s.HeadroomCacheRequested[host]
	if exists {
		fmt.Sprintf("Query host %s with bw %f", host, headroomReq)
		bwResponse, err := s.QueryHeadroom(host, float32(headroomReq.SendBw))
		if err == nil {

			bwInfo := GetBwResults(bwResponse)
			if len(bwInfo.BandwidthResults) > 0 {
				log.Printf("Headroom host %s requested = %f actual = %f\n", host, headroomReq.SendBw, bwInfo.BandwidthResults[0].RcvBw)
				headroomInfo.BandwidthResults = append(headroomInfo.BandwidthResults, bwInfo.BandwidthResults[0])
				s.headroomIdx = (s.headroomIdx + 1) % len(s.hosts)
			}
		}
	}
	//}
	trResponse, err := s.QueryTrStats()
	if err == nil {
		trInfo = GetTrResults(trResponse)
	}
	return headroomInfo, trInfo
}

func (s *server) UpdateCache() {
	s.mu.Lock()
	if s.pendingBwRequest {
		bwInfo, _, latencyInfo := s.GetUpdatedNetStats()

		for _, bwResult := range bwInfo.BandwidthResults {
			log.Printf("Updated %s", bwResult.Host)
			s.BwCache[bwResult.Host] = bwResult
		}
		if s.hostIdx == len(s.hosts) {
			s.pendingBwRequest = false
			s.hostIdx = 0
		}
		s.LatencyCache = latencyInfo
	}
	bwInfo, trInfo := s.GetUpdatedHeadroomStats()
	for _, bwResult := range bwInfo.BandwidthResults {
		fmt.Printf("Updated %s headroom", bwResult.Host)
		s.HeadroomCacheMeasured[bwResult.Host] = bwResult
	}
	if len(trInfo.TracerouteResults) > 0 {
		s.TrCache = trInfo
	}

	s.mu.Unlock()
}

func (s *server) DoInBackground() {
	//go func() {
	for {
		s.UpdateCache()
		s.bpfRunner.GetStats()
		time.Sleep(15 * time.Second)
	}
	//}()
}

func StartServer() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	hosts := readConfig(*config)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	bpfRunner := NewBPFRunner(*device)
	s := grpc.NewServer()
	client := http.Client{Timeout: 60 * time.Second}
	monserver := &server{netClient: client, hosts: hosts, bpfRunner: bpfRunner, hostIdx: 0, BwCache: make(map[string]Bandwidth, 0), HeadroomCacheRequested: make(map[string]pb.BandwidthInfo, 0), HeadroomCacheMeasured: make(map[string]Bandwidth, 0), pendingBwRequest: true, headroomIdx: 0}

	pb.RegisterNetMonitorServer(s, monserver)
	log.Printf("server listening at %v", lis.Addr())
	go func() {
		monserver.DoInBackground()
	}()
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func main() {
	f, err := os.OpenFile("netmon_log", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	log.SetOutput(f)
	StartServer()
}
