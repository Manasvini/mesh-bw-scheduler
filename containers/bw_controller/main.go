package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"

	bw_controller "github.gatech.edu/cs-epl/mesh-bw-scheduler/bwcontroller"
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
	kubeClient := bw_controller.NewKubeClient(config.KubeProxyAddr, config.KubeNodesEndpoint, config.KubePodsEndpoint, config.KubeNamespaces)
	netmonClient := bw_controller.NewNetmonClient(config.NetmonAddrs)
	controller := bw_controller.NewController(promClient, netmonClient, kubeClient)
	controller.UpdateNodes()
	controller.UpdatePods()
	controller.UpdatePodMetrics()
	controller.UpdateNetMetrics()
	controller.EvaluateDeployment()
}
