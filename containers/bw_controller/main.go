package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"time"
)

var (
	prometheus string        = "http://0.0.0.0:9090/"
	timeout    time.Duration = time.Second * 30
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
	controller := NewController(config.PromAddr, config.NetmonAddrs, config.PromMetrics)
	controller.UpdatePodMetrics()
	controller.UpdateNodeMetrics()
}
