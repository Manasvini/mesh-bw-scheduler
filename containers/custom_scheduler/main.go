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
	addrs             = []string{"192.168.160.42:50051"}
)

const schedulerName = "epl-scheduler"

func main() {
	logger("Starting epl scheduler...")
	ns := []string{"epl"}
	client := KubeClient{apiHost: apiHost,
		bindingsEndpoint:  bindingsEndpoint,
		eventsEndpoint:    eventsEndpoint,
		nodesEndpoint:     nodesEndpoint,
		podsEndpoint:      podsEndpoint,
		watchPodsEndpoint: watchPodsEndpoint,
		metricsEndpoint:   metricsEndpoint,
		configEndpoint:    configEndpoint,
		namespaces:        ns}

	done := client.WaitForProxy()

	dagSched := &DagScheduler{client: &client, processorLock: &sync.Mutex{}, podProcessor: NewPodProcessor(), netmonClient: netmon_client.NewNetmonClient(addrs)}
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
	go dagSched.ReconcileUnscheduledPods(30, doneChan, &wg)

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
