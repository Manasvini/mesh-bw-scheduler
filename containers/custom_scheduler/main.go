// Copyright 2016 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	netmon_client "github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon_client"
)

var (
	apiHost           = "127.0.0.1:8001"
	bindingsEndpoint  = "/api/v1/namespaces/%s/pods/%s/binding/"
	eventsEndpoint    = "/api/v1/watch/namespaces/%s/events"
	nodesEndpoint     = "/api/v1/nodes"
	podsEndpoint      = "/api/v1/namespaces/%s/pods/"
	watchPodsEndpoint = "/api/v1/watch/namespaces/%s/pods"
	configEndpoint    = "/apis/apps/v1/namespaces/epl/deployments/epl-scheduler"
	metricsEndpoint   = "/apis/metrics.k8s.io/v1beta1/nodes"
	addrs             = []string{"localhost:50051"}
)

const schedulerName = "epl-scheduler"

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
	f, err := os.OpenFile("sched_log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}

	//defer to close when you're done with it, not because you think it's idiomatic!
	defer f.Close()

	//set output of logs to f
	log.SetOutput(f)
	logger("Starting epl scheduler...")
	var configFile string
	flag.StringVar(&configFile, "config", "./config.json", "Config file path")
	flag.Parse()
	config := parseConfig(configFile)
	client := KubeClient{apiHost: config.ApiHost,
		bindingsEndpoint:  config.BindingsEndpoint,
		eventsEndpoint:    config.EventsEndpoint,
		nodesEndpoint:     config.NodesEndpoint,
		podsEndpoint:      config.PodsEndpoint,
		watchPodsEndpoint: config.WatchPodsEndpoint,
		metricsEndpoint:   config.MetricsEndpoint,
		configEndpoint:    config.ConfigEndpoint,
		namespaces:        config.Namespaces}

	done := client.WaitForProxy()
	logger(fmt.Sprintf("Got %d namespaces", len(config.Namespaces)))
	dagSched := &DagScheduler{client: &client, processorLock: &sync.Mutex{}, podProcessor: NewPodProcessor(&client), netmonClient: netmon_client.NewNetmonClient(config.NetmonAddrs)}
	if done == 0 {
		logger("Failed to connect to proxy.")
		os.Exit(0)
	}

	logger("Succeeded to connect to proxy.")

	doneChan := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go dagSched.MonitorUnscheduledPods(doneChan, &wg)

	wg.Add(1)
	go dagSched.ReconcileUnscheduledPods(20, doneChan, &wg)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case <-signalChan:
			logger("Shutdown signal received, exiting...")
			close(doneChan)
			wg.Wait()
			os.Exit(0)
		}
	}
}
