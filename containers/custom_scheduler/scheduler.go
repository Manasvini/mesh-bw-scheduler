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
	"strconv"
	"strings"
	"sync"
	"time"

	netmon_client "github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon_client"
	"k8s.io/apimachinery/pkg/api/resource"
)

type DagScheduler struct {
	client        *KubeClient
	netmonClient  *netmon_client.NetmonClient
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
	nodes, err := sched.FitPod(pod)
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

func (sched *DagScheduler) getNodeResourcesRemaining(nodeList *NodeList, nodeMetrics *NodeMetricsList) map[string]Resource {
	nodeResources := make(map[string]Resource, 0)
	for _, node := range nodeList.Items {
		nodeResource := Resource{cpu: 0, memory: 0}
		cpu := node.Status.Allocatable["cpu"]
		memory := node.Status.Allocatable["memory"]
		resCpu := resource.MustParse(cpu)
		resMem := resource.MustParse(memory)
		nodeResource.cpu = resCpu.Value()
		nodeResource.memory = resMem.Value()
		nodeResource.name = node.Metadata.Name
		nodeResources[node.Metadata.Name] = nodeResource

	}
	for _, node := range nodeMetrics.Items {
		nodeResource, exists := nodeResources[node.Metadata.Name]
		cpu := node.Usage.Cpu
		memory := node.Usage.Memory
		if exists {
			resCpu := resource.MustParse(cpu)
			resMem := resource.MustParse(memory)
			nodeResource.cpu -= resCpu.Value()
			nodeResource.memory -= resMem.Value()
		}
		nodeResources[node.Metadata.Name] = nodeResource
	}

	return nodeResources
}

func (sched *DagScheduler) getNetResourcesRemaining(paths netmon_client.PathSet, traffics netmon_client.TrafficSet) netmon_client.PathSet {
	availableCap := make(netmon_client.PathSet, 0)
	for src, pdsts := range paths {
		p, exists := availableCap[src]
		if !exists {
			p = make(map[string]netmon_client.Path, 0)
			availableCap[src] = p
		}
		for dst, path := range pdsts {
			availableCap[src][dst] = path
		}
	}

	for src, dstTrafs := range traffics {
		trafs, exists := availableCap[src]
		if !exists {
			continue
		}
		for dst, traf := range dstTrafs {
			bwInfo, exists := trafs[dst]
			if !exists {
				continue
			}
			bwInfo.Bandwidth -= traf.Bytes
			trafs[dst] = bwInfo
		}
	}
	return availableCap
}

func (sched *DagScheduler) Fit(pod Pod, node Node,
	nodeResource Resource,
	availableBw netmon_client.PathSet) bool {
	podResource := Resource{cpu: 0, memory: 0}
	for _, container := range pod.Spec.Containers {
		containerCpu, exists := container.Resources.Requests["cpu"]

		if exists {
			cpuRes := resource.MustParse(containerCpu)
			podResource.cpu += cpuRes.Value()
		}
		containerMemory, exists := container.Resources.Requests["memory"]
		if exists {
			memRes := resource.MustParse(containerMemory)
			podResource.memory += memRes.Value()
		}
	}
	podBw := 0.0
	for k, v := range pod.Metadata.Annotations {
		vals := strings.Split(k, ".")
		if "bw" == vals[0] {
			bw, _ := strconv.Atoi(v)
			podBw += float64(bw)
		}
	}

	nodeBw := 0.0
	nodeIp, _ := node.Metadata.Annotations["flannel.alpha.coreos.com/public-ip"]
	nodeBws, exists := availableBw[nodeIp]
	if !exists {
		logger("Error: No bw info for node " + node.Metadata.Name)
		return false
	}
	for _, bw := range nodeBws {
		nodeBw += bw.Bandwidth
	}

	logger(fmt.Sprintf("Pod %s cpu = %f memory = %f bw = %f  Node %s cpu = %f memory = %f bw = %f", pod.Metadata.Name, podResource.cpu, podResource.memory, podBw, node.Metadata.Name, nodeResource.cpu, nodeResource.memory, nodeBw))
	if podResource.cpu < nodeResource.cpu {
		logger(fmt.Sprintf("pod %s node %s insufficient CPU"))
		return false
	}
	if podResource.memory < nodeResource.memory {
		logger(fmt.Sprintf("pod %s node %s insufficient memory"))
		return false
	}
	if podBw < nodeBw {
		logger(fmt.Sprintf("pod %s node %s insufficient bw"))
		return false
	}
	return true

}

func (sched *DagScheduler) AreDepsSatisfied(currentPod Pod, currentNode Node,
	assignments map[string]string,
	availableBws netmon_client.PathSet) bool {
	for k, v := range currentPod.Metadata.Annotations {
		vals := strings.Split(k, ".")
		qtyName, podName := vals[0], vals[1]
		bw := 0.0
		if qtyName == "bw" {
			bwVal, _ := strconv.Atoi(v)
			bw = float64(bwVal)
		} else {
			continue
		}
		nodeIp, _ := currentNode.Metadata.Annotations["flannel.alpha.coreos.com/public-ip"]
		nodeBws, exists := availableBws[nodeIp]
		dstNode, exists := assignments[podName]
		if !exists {
			continue
		}

		path, dExists := nodeBws[dstNode]
		if !dExists {
			return false
		}
		if bw > path.Bandwidth {
			return false
		}

	}
	return true
}

func getNodeWithName(nodeName string, nodes *NodeList) Node {
	var node Node
	for _, n := range nodes.Items {
		if n.Metadata.Name == nodeName {
			return n
		}
	}
	return node
}

func (sched *DagScheduler) SchedulePods() error {
	sched.processorLock.Lock()
	defer sched.processorLock.Unlock()
	// returns the dag of unscheduled pods
	pods, podGraph := sched.podProcessor.GetUnscheduledPods()
	logger(fmt.Sprintf("got %d pods", len(pods)))

	_, paths, traffics := sched.netmonClient.GetStats()

	nodes, _ := sched.client.GetNodes()
	nodeMetrics, _ := sched.client.GetNodeMetrics()

	nodeResources := sched.getNodeResourcesRemaining(nodes, nodeMetrics)
	netResources := sched.getNetResourcesRemaining(paths, traffics)

	logger(fmt.Sprintf("got %d paths and %d traffics", len(paths), len(traffics)))
	topoOrder := topoSort(podGraph)

	nodeResList := make([]Resource, 0)
	for _, nr := range nodeResources {
		nodeResList = append(nodeResList, nr)
	}
	sortNodes(nodeResList)

	podToSchedule := topoOrder[0]
	scheduledPods := make(map[string]bool, 0)
	podAssignment := make(map[string]string, 0)
	madeAssignment := false
	candidateNodeIdx := 0
	candidateNodeRes := nodeResList[candidateNodeIdx]
	candidateNode := getNodeWithName(candidateNodeRes.name, nodes)

	for {
		if len(scheduledPods) == len(pods) {
			break
		}
		if madeAssignment {
			sortNodes(nodeResList)
		}

		if sched.Fit(pods[podToSchedule], candidateNode, candidateNodeRes, netResources) &&
			sched.AreDepsSatisfied(pods[podToSchedule], candidateNode,
				podAssignment, netResources) {
			logger(fmt.Sprintf("Found node %s for pod %s", candidateNode.Metadata.Name, podToSchedule))
		}

	}

	return nil
}

// TODO fix Fit() to check if topology constraints are met
func (sched *DagScheduler) FitPod(pod *Pod) ([]Node, error) {

	//	nodeList, err := sched.client.GetNodes()
	//
	//	if err != nil {
	//		logger(err)
	//		return nil, err
	//	}
	//	logger(fmt.Sprintf("Got %d nodes", len(nodeList.Items)))
	//	config, err := sched.client.GetConfig()
	//	if err != nil {
	//		logger(err)
	//		return nil, err
	//	}
	//
	//	logger("Found config: " + string(config))
	//
	var nodes []Node
	//
	//	for _, node := range nodeList.Items {
	//		if node.Metadata.Name == config {
	//			nodes = append(nodes, node)
	//		}
	//	}
	//
	//	if (len(nodes) == 0) || (config == "") {
	//		logger("Failed to schedule pod " + pod.Metadata.Name)
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
	//	}

	return nodes, nil
}
