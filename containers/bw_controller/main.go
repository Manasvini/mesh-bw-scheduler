package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	bw_controller "github.gatech.edu/cs-epl/mesh-bw-scheduler/bwcontroller"
	netmon_client "github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon_client"
)

func parseConfig(filename string) Config {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}

	// Now let's unmarshall the data into `payload`
	var payload Config
	err = json.Unmarshal(content, &payload)
	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
	}
	return payload
}

func parseIpMap(filename string) map[string]string {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}

	// Now let's unmarshall the data into `payload`
	var payload netmon_client.NodeMap 
	err = json.Unmarshal(content, &payload)
	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
	}
	mappings := make(map[string]string, 0)
	for _, mapping := range payload.Mappings {
		mappings[mapping.Src] = mapping.Dst
	}
	return mappings

}
func main() {
	f, err := os.OpenFile("controller_log", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	log.SetOutput(f)
	var configFile string
	var bwInfoFile string
	var migrationInfoFile string
	var ipMapFile string
	flag.StringVar(&configFile, "config", "./config.json", "Config file path (input)")
	flag.StringVar(&bwInfoFile, "bw", "./bwinfo.csv", "bw info file (output)")
	flag.StringVar(&migrationInfoFile, "migration", "./migration.csv", "migration info file(output)")
	flag.StringVar(&ipMapFile, "ipmap", "./nodemap.json", "IP map file path")
	
	flag.Parse()
	config := parseConfig(configFile)
	ipMap := parseIpMap(ipMapFile)
	promClient := bw_controller.NewPrometheusClient(config.PromAddr, config.PromMetrics)
	kubeClient := bw_controller.NewKubeClient(config.KubeProxyAddr, config.KubeNodesEndpoint, config.KubePodsEndpoint, config.KubeDeleteEndpoint, config.KubeNamespaces)
	netmonClient := netmon_client.NewNetmonClient(config.NetmonAddrs)
	controller := bw_controller.NewController(promClient, netmonClient, kubeClient, config.ValuationInterval, config.UtilChangeThreshold, bwInfoFile, migrationInfoFile, config.HeadroomThreshold, ipMap)

	monCh := controller.MonitorState(time.Duration(config.MonDurationSeconds) * time.Second)
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	sig := <-signalChannel
	switch sig {
	case os.Interrupt:
		monCh <- true
	case syscall.SIGTERM:
		//handle SIGTERM
		monCh <- true
	
	}
	controller.Shutdown()
}
