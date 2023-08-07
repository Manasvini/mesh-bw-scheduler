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

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "./config.json", "Config file path")
	config := parseConfig(configFile)
	promClient := bw_controller.NewPrometheusClient(config.PromAddr, config.PromMetrics)
	kubeClient := bw_controller.NewKubeClient(config.KubeProxyAddr, config.KubeNodesEndpoint, config.KubePodsEndpoint, config.KubeDeleteEndpoint, config.KubeNamespaces)
	netmonClient := netmon_client.NewNetmonClient(config.NetmonAddrs)
	controller := bw_controller.NewController(promClient, netmonClient, kubeClient)

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
}
