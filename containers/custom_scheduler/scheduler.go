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
	"fmt"
	"sync"
	"time"
)

type DagScheduler struct {
	client        *KubeClient
	podProcessor  *PodProcessor
	processorLock *sync.Mutex
}

func (sched *DagScheduler) ReconcileUnscheduledPods(interval int, done chan struct{}, wg *sync.WaitGroup) {
	for {
		select {
		case <-time.After(time.Duration(interval) * time.Second):
			err := sched.SchedulePods()
			if err != nil {
				logger(err)
			}
		case <-done:
			wg.Done()
			logger("Stopped reconciliation loop.")
			return
		}
	}
}

func (sched *DagScheduler) MonitorUnscheduledPods(done chan struct{}, wg *sync.WaitGroup) {
	pods, errc := sched.client.WatchUnscheduledPods()

	for {
		select {
		case err := <-errc:
			logger(err)
		case pod := <-pods:
			sched.processorLock.Lock()

			time.Sleep(2 * time.Second)
			//err := schedulePod(&pod)
			// add the pod to pod processor
			// pod processor collects pod and builds the pod DAG
			sched.podProcessor.AddPod(pod)
			// TODO call SchedulePods()
			//if err != nil {
			//	logger(err)
			//}
			sched.processorLock.Unlock()
		case <-done:
			wg.Done()
			logger("Stopped scheduler.")
			return
		}
	}
}

func (sched *DagScheduler) SchedulePod(pod *Pod) error {
	nodes, err := sched.Fit(pod)
	if err != nil {
		logger(err)
		return err
	}
	if len(nodes) == 0 {
		return fmt.Errorf("Unable to schedule pod (%s) failed to fit in any node", pod.Metadata.Name)
	}
	err = sched.client.Bind(pod, nodes[0])
	if err != nil {
		logger(err)
		return err
	}
	return nil
}

func (sched *DagScheduler) SchedulePods() error {
	sched.processorLock.Lock()
	defer sched.processorLock.Unlock()
	// returns the dag of unscheduled pods
	pods := sched.podProcessor.GetUnscheduledPods()

	// TODO Add code to perform topo sort etc
	logger(fmt.Sprintf("got %d pods", len(pods)))
	for _, pod := range pods {
		err := sched.SchedulePod(&pod)
		if err != nil {
			logger(err)
		}
	}
	return nil
}

// TODO fix Fit() to check if topology constraints are met
func (sched *DagScheduler) Fit(pod *Pod) ([]Node, error) {
	nodeList, err := sched.client.GetNodes()
	if err != nil {
		logger(err)
		return nil, err
	}
	logger(fmt.Sprintf("Got %d nodes", len(nodeList.Items)))
	config, err := sched.client.GetConfig()
	if err != nil {
		logger(err)
		return nil, err
	}

	logger("Found config: " + string(config))

	var nodes []Node

	for _, node := range nodeList.Items {
		if node.Metadata.Name == config {
			nodes = append(nodes, node)
		}
	}

	if (len(nodes) == 0) || (config == "") {
		logger("Failed to schedule pod " + pod.Metadata.Name)
		// Emit a Kubernetes event that the Pod failed to be scheduled.
		// timestamp := time.Now().UTC().Format(time.RFC3339)
		// event := Event{
		// 	Count:          1,
		// 	Message:        fmt.Sprintf("pod (%s) failed to fit in any node\n", pod.Metadata.Name),
		// 	Reason:         "FailedScheduling",
		// 	LastTimestamp:  timestamp,
		// 	FirstTimestamp: timestamp,
		// 	Type:           "Warning",
		// 	Source:         EventSource{Component: "epl-scheduler"},
		// 	InvolvedObject: ObjectReference{
		// 		Kind:      "Pod",
		// 		Name:      pod.Metadata.Name,
		// 		Namespace: "epl",
		// 		Uid:       pod.Metadata.Uid,
		// 	},
		// }

		// postEvent(event)
	}

	return nodes, nil
}
