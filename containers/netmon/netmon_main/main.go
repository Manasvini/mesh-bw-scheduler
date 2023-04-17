package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	pb "github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon/proto"
	"google.golang.org/grpc"
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

func readConfig(configfile string) []string {
	body, err := ioutil.ReadFile(configfile)
	if err != nil {
		log.Fatalf("unable to read file: %v", err)
	}
	hosts := strings.Split(string(body), "\n")
	return hosts
}

type server struct {
	pb.UnimplementedNetMonitorServer
	BwCache   BandwidthResults  // dest node -> bw map
	TrCache   TracerouteResults // dest node -> traceroute map
	mu        sync.Mutex
	netClient http.Client
	hosts     []string
	bpfRunner *BPFRunner
}

func (s *server) QueryBwStats(hostname string) []byte {
	reqURL := "http://0.0.0.0:6000"
	res, err := s.netClient.Get(reqURL + "/bw?host=" + hostname)
	if err != nil {
		fmt.Printf("client: could not create request: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("client: status code: %d\n", res.StatusCode)

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("client: could not read response body: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf(string(resBody))
	return resBody

}

func (s *server) QueryTrStats() []byte {
	reqURL := "http://0.0.0.0:6000"
	res, err := s.netClient.Get(reqURL + "/traceroute")
	if err != nil {
		fmt.Printf("client: could not create request: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("client: status code: %d\n", res.StatusCode)

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("client: could not read response body: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf(string(resBody))
	return resBody
}

func GetBwResults(bwResponse []byte) BandwidthResults {
	var bwInfo BandwidthResults
	json.Unmarshal(bwResponse, &bwInfo)
	return bwInfo
}

func GetTrResults(trResponse []byte) TracerouteResults {
	var trInfo TracerouteResults
	json.Unmarshal(trResponse, &trInfo)
	return trInfo
}

func (s *server) GetNetInfo(ctx context.Context, in *pb.NetInfoRequest) (*pb.NetInfoReply, error) {
	log.Printf("Received: req")
	s.mu.Lock()
	bwInfos := make([]*pb.BandwidthInfo, 0)
	for _, bw := range s.BwCache.BandwidthResults {
		bwInfo := pb.BandwidthInfo{Host: bw.Host, SendBw: float32(bw.Snd), ReceiveBw: float32(bw.Rcv)}
		bwInfos = append(bwInfos, &bwInfo)
	}
	trInfos := make([]*pb.TracerouteInfo, 0)
	for _, tr := range s.TrCache.TracerouteResults {
		trInfo := pb.TracerouteInfo{Host: tr.Host, Hops: tr.Route}
		trInfos = append(trInfos, &trInfo)
	}
	reply := &pb.NetInfoReply{BwInfo: bwInfos, TrInfo: trInfos}
	s.mu.Unlock()
	return reply, nil
}

func (s *server) GetUpdatedNetStats() (BandwidthResults, TracerouteResults) {
	var allBwInfo BandwidthResults
	for _, host := range s.hosts {
		bwResponse := s.QueryBwStats(host)
		bwInfo := GetBwResults(bwResponse)
		if len(bwInfo.BandwidthResults) == 0 {
			continue
		}
		allBwInfo.BandwidthResults = append(allBwInfo.BandwidthResults, bwInfo.BandwidthResults[0])
	}
	trResponse := s.QueryTrStats()

	trInfo := GetTrResults(trResponse)
	fmt.Printf("Got %d hosts\n", len(trInfo.TracerouteResults))
	fmt.Printf("Got %d hosts\n", len(allBwInfo.BandwidthResults))
	return allBwInfo, trInfo
}

func (s *server) UpdateCache() {
	bwInfo, trInfo := s.GetUpdatedNetStats()
	s.mu.Lock()
	s.BwCache = bwInfo
	s.TrCache = trInfo
	s.mu.Unlock()
}

func (s *server) DoInBackground() {
	go func() {
		for {
			s.UpdateCache()
			s.bpfRunner.PrintStats()
			time.Sleep(300 * time.Second)
		}
	}()
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
	client := http.Client{Timeout: 300 * time.Second}
	monserver := &server{netClient: client, hosts: hosts, bpfRunner: bpfRunner}

	pb.RegisterNetMonitorServer(s, monserver)
	log.Printf("server listening at %v", lis.Addr())
	monserver.DoInBackground()
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func main() {
	StartServer()
}
