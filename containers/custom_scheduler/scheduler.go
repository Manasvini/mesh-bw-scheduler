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
	client        KubeClientIntf
	netmonClient  netmon_client.NetmonClientIntf
	podProcessor  *PodProcessor
	processorLock *sync.Mutex
}

func (sched *DagScheduler) ReconcileUnscheduledPods(interval int, done chan struct{}, wg *sync.WaitGroup) {
	for {
		select {
		case <-time.After(time.Duration(interval) * time.Second):
			for {
				pods, podGraph := sched.podProcessor.GetUnscheduledPods()
				if len(pods) == 0 {
					break
				}
				assignment, pods, nodes := sched.SchedulePods(pods, podGraph)
				if len(pods) == 0 {
					logger("Could not schedule any pod")
					break
				}
				err := sched.AssignPods(assignment, pods, nodes)
				if err != nil {
					logger(err)
				}

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
	logger("monitoring unscheduled pods")
	for {
		select {
		case err := <-errc:
			logger(err)
		case pod := <-pods:
			sched.processorLock.Lock()
			logger(fmt.Sprintf("Got pod %s", pod.Metadata.Name))
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

func (sched *DagScheduler) SchedulePod(pod Pod, node Node) error {
	err := sched.client.Bind(pod, node)
	pods := []Pod{pod}
	sched.podProcessor.MarkScheduled(pods)
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
			logger(fmt.Sprintf("src %s dst %s available %f traf %f", src, dst, bwInfo.Bandwidth, traf.Bytes))
			bwInfo.Bandwidth -= traf.Bytes
			trafs[dst] = bwInfo
		}
	}
	return availableCap
}

func (sched *DagScheduler) GetPodResource(pod Pod) Resource {
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
	return podResource

}
func (sched *DagScheduler) EvalPredicate(pod Pod, node Node, availableBw netmon_client.PathSet) bool {
	nodeIp, _ := node.Metadata.Annotations["flannel.alpha.coreos.com/public-ip"]
	nodeBws, exists := availableBw[nodeIp]
	if !exists {
		logger("Error: No bw info for node " + node.Metadata.Name)
		return false
	}
	annotations := pod.Metadata.Annotations
	for k, v := range annotations {
		vals := strings.Split(k, ".")
		if "neighbor" == vals[0] && "bw" == vals[2] {
			if "all" == vals[1] {
				for _, bw := range nodeBws {
					req, _ := strconv.Atoi(v)
					if bw.Bandwidth < float64(req) {
						return false
					}
				}
			} else if "any" == vals[1] {
				found := false
				for _, bw := range nodeBws {
					req, _ := strconv.Atoi(v)
					if bw.Bandwidth >= float64(req) {
						found = true
					}
				}
				if found == false {
					return false
				}
			}
		}
	}
	return true
}

func (sched *DagScheduler) Fit(pod Pod, node Node,
	nodeResource Resource,
	availableBw netmon_client.PathSet) bool {
	podResource := sched.GetPodResource(pod)
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
	logger(fmt.Sprintf("node name %s ip %s", node.Metadata.Name, nodeIp))
	nodeBws, exists := availableBw[nodeIp]
	if !exists {
		logger("Error: No bw info for node " + node.Metadata.Name)
		return false
	}
	for dst, bw := range nodeBws {
		nodeBw += bw.Bandwidth
		logger(fmt.Sprintf("add dst %s bw %f", dst, bw.Bandwidth))
	}

	if !sched.EvalPredicate(pod, node, availableBw) {
		logger("node " + node.Metadata.Name + " does not have sufficient bw for predicate")
		return false
	}

	logger(fmt.Sprintf("Pod %s cpu = %d memory = %d bw = %f  Node %s cpu = %d memory = %d bw = %f", pod.Metadata.Name, podResource.cpu, podResource.memory, podBw, node.Metadata.Name, nodeResource.cpu, nodeResource.memory, nodeBw))
	if podResource.cpu > nodeResource.cpu {
		logger(fmt.Sprintf("pod %s node %s insufficient CPU", pod.Metadata.Name, node.Metadata.Name))
		return false
	}
	if podResource.memory > nodeResource.memory {
		logger(fmt.Sprintf("pod %s node %s insufficient memory", pod.Metadata.Name, node.Metadata.Name))
		return false
	}
	if podBw > nodeBw {
		logger(fmt.Sprintf("pod %s node %s insufficient bw", pod.Metadata.Name, node.Metadata.Name))
		return false
	}
	return true

}

func (sched *DagScheduler) AreDepsSatisfied(currentPod Pod, currentNode Node, nodes *NodeList,
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
		logger(fmt.Sprintf("pod %s -> %s needs %f", currentPod.Metadata.Name, podName, bw))
		nodeIp, _ := currentNode.Metadata.Annotations["flannel.alpha.coreos.com/public-ip"]
		nodeBws, exists := availableBws[nodeIp]
		dstNode, exists := assignments[podName]
		if !exists {
			continue
		}
		dstNodeInfo := getNodeWithName(dstNode, nodes)
		dstNodeIp := dstNodeInfo.Metadata.Annotations["flannel.alpha.coreos.com/public-ip"]

		path, dExists := nodeBws[dstNodeIp]
		if !dExists {
			return false
		}
		if bw > path.Bandwidth {
			return false
		}

	}
	return true
}

func (sched *DagScheduler) getNextPod(currentPod string, assignedPods map[string]string, topoOrder []string, podGraph map[string]map[string]bool) ([]string, bool) {
	neighbors := getNeighbors(currentPod, podGraph)
	logger("current pod = " + currentPod)
	allK8sPods, _ := sched.client.GetPods()
	logger(fmt.Sprintf("%s has %d neighbors\n", currentPod, len(neighbors)))
	if len(neighbors) > 0 {
		unscheduled := make([]string, 0)
		for _, n := range neighbors {
			_, exists := assignedPods[n]
			if n != currentPod && !exists && !sched.podProcessor.IsPodInList(allK8sPods, currentPod) {

				unscheduled = append(unscheduled, n)
				logger("pod " + n + " is unassigned")
			}
		}
		if len(unscheduled) > 0 {
			return unscheduled, true
		}
	}
	for _, p := range topoOrder {

		exists := false
		for pname, _ := range assignedPods {
			if getPodName(pname) == p {
				exists = true
			}
		}
		if !exists && !sched.podProcessor.IsPodInList(allK8sPods, p) && p != currentPod {
			return []string{p}, true
		}
	}
	return []string{""}, false
}

func getNodeWithName(nodeName string, nodes *NodeList) Node {
	var node Node
	logger(fmt.Sprintf("Got %d nodes", len(nodes.Items)))
	for _, n := range nodes.Items {

		logger("node name = " + nodeName + " from meta = " + n.Metadata.Name)
		if n.Metadata.Name == nodeName {

			return n
		}
	}
	return node
}

/*func getPodWithName(podName string, pods map[string]Pod) Pod {
	logger("find pod for name " + podName)
	var pod Pod
	for p, podInfo := range pods {
		logger("Pod is " + p)
		if getPodName(p) == podName {
			return podInfo
		}
	}
	return pod

}*/

func (sched *DagScheduler) SchedulePods(pods map[string]Pod, podGraph map[string]map[string]bool) (map[string]string, map[string]Pod, *NodeList) {
	// returns the dag of unscheduled pods
	sched.processorLock.Lock()
	defer sched.processorLock.Unlock()
	//pods, podGraph := sched.podProcessor.GetUnscheduledPods()
	logger(fmt.Sprintf("got %d pods and %d podgraph", len(pods), len(podGraph)))

	_, paths, traffics := sched.netmonClient.GetStats()

	nodes, _ := sched.client.GetNodes()
	nodeMetrics, _ := sched.client.GetNodeMetrics()
	logger(fmt.Sprintf("Got %d nodes", len(nodes.Items)))
	podAssignment := make(map[string]string, 0)
	if len(nodes.Items) == 0 {
		logger("ERROR: Cannot find any node for scheduling, skipping")
		return podAssignment, pods, nodes
	}
	nodeResources := sched.getNodeResourcesRemaining(nodes, nodeMetrics)
	netResources := sched.getNetResourcesRemaining(paths, traffics)

	logger(fmt.Sprintf("got %d paths and %d traffics", len(paths), len(traffics)))
	topoOrder := topoSortWithChain(podGraph)
	logger(fmt.Sprintf("topo order has %d pods", len(topoOrder)))
	nodeResList := make([]Resource, 0)
	for _, nr := range nodeResources {
		nodeResList = append(nodeResList, nr)
	}
	sortNodes(nodeResList)
	podIdx := 0
	madeAssignment := false

	if len(topoOrder) == 0 {
		logger("No pods to schedule..")

		return podAssignment, pods, nodes
	}
	allPods, _ := sched.client.GetPods()
	candidateNodeIdx := 0
	//#unscheduledNeighbors := make([]string, 0)
	podsToSchedule := make([]string, 0)
	for _, p := range topoOrder {
		if !sched.podProcessor.IsPodInList(allPods, p) {
			podsToSchedule = append(podsToSchedule, p)
			logger("topo order now has " + p)
		}
	}
	topoOrder = podsToSchedule
	if len(topoOrder) == 0 {
		logger("No pods to schedule..")

		return podAssignment, pods, nodes
	}
	podToSchedule := topoOrder[podIdx]
	for {
		logger(fmt.Sprintf("Have %d pods to schedule candidate idx = %d", len(topoOrder)-len(podAssignment), candidateNodeIdx))
		if len(podAssignment) == len(topoOrder) {
			break
		}
		if candidateNodeIdx == len(nodeResList) {
			break
		}
		if madeAssignment {
			sortNodes(nodeResList)
			candidateNodeIdx = 0
			// TODO do something smarter here, like processing neighbor pods instead of topo sorted pods
			podIdx += 1
			if podIdx == len(topoOrder) {
				break
			}
			podToSchedule = topoOrder[podIdx]
			madeAssignment = false
		}
		logger(fmt.Sprintf("Assign pod %s", podToSchedule))
		candidateNodeRes := nodeResList[candidateNodeIdx]
		candidateNode := getNodeWithName(candidateNodeRes.name, nodes)
		podMeta := getPodWithName(podToSchedule, pods)
		if podMeta.Metadata.Name == "" {
			logger("pod for " + podToSchedule + " does not exist")
			break
		}
		if sched.Fit(podMeta, candidateNode, candidateNodeRes, netResources) && sched.AreDepsSatisfied(pods[podToSchedule], candidateNode, nodes,
			podAssignment, netResources) {
			logger(fmt.Sprintf("Found node %s for pod %s meta =%s", candidateNode.Metadata.Name, podToSchedule, podMeta.Metadata.Name))
			podAssignment[podMeta.Metadata.Name] = candidateNode.Metadata.Name
			podResource := sched.GetPodResource(pods[podToSchedule])
			candidateNodeRes.cpu -= podResource.cpu
			candidateNodeRes.memory -= podResource.memory
			nodeResources[candidateNodeRes.name] = candidateNodeRes
			nodeResList[candidateNodeIdx] = candidateNodeRes
			madeAssignment = true

		} else {
			logger(fmt.Sprintf("%s does not fit on %s", podToSchedule, candidateNodeRes.name))
			candidateNodeIdx += 1
			madeAssignment = false
		}
	}
	return podAssignment, pods, nodes
}

func (sched *DagScheduler) AssignPods(podAssignment map[string]string, pods map[string]Pod, nodes *NodeList) error {
	for pod, nodeName := range podAssignment {
		node := getNodeWithName(nodeName, nodes)
		logger("pod = " + pod + " pod name= " + pods[pod].Metadata.Name)
		logger("Assign pod " + pods[pod].Metadata.Name + " to node " + node.Metadata.Name)
		err := sched.SchedulePod(pods[pod], node)
		if err != nil {
			logger(fmt.Sprintf("Got error %v", err))
		}
	}
	return nil
}
