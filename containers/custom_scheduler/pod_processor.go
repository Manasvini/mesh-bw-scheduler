package main

import (
	"fmt"
	"strings"
	"sync"
)

//var processorLock = &sync.Mutex{}

type PodProcessor struct {
	unscheduledPods map[string]Pod // pod name to pod mapping
	podLock         *sync.Mutex
}

func NewPodProcessor() *PodProcessor {
	mu := &sync.Mutex{}
	unscheduledPods := make(map[string]Pod, 0)
	pp := &PodProcessor{unscheduledPods: unscheduledPods, podLock: mu}
	logger("Created pod processor")
	return pp
}

func (pp *PodProcessor) AddPod(pod Pod) {
	pp.podLock.Lock()
	pp.unscheduledPods[pod.Metadata.Name] = pod

	pp.podLock.Unlock()
	logger("Added pod " + pod.Metadata.Name + " to unscheduled pods")
}

func (pp *PodProcessor) AreAllRelatedPodsPresent(pod Pod, relationship string) bool {
	// Dependson: for a pod, check if all  the pods that THIS pod depends on are present
	// Dependedby: for a pod, check if all pods that depend on THIS pod are present
	pp.podLock.Lock()
	podList := pp.unscheduledPods
	pp.podLock.Unlock()
	annotations := pod.Metadata.Annotations
	// format: dependedby.PODNAME
	for k, _ := range annotations {
		if strings.Contains(k, relationship) {
			vals := strings.Split(k, ".")
			if len(vals) < 2 {
				logger(fmt.Sprintf("ERROR: Incorrect annotation format for pod dependency %s", k))
			}
			if relationship != vals[0] {
				continue
			}
			podName := vals[1]
			_, isPodPresent := podList[podName]
			if !isPodPresent {
				return false
			}
		}
	}
	return true
}

func (pp *PodProcessor) AreAllDependersPresent(pod Pod) bool {
	return pp.AreAllRelatedPodsPresent(pod, "dependedby")
}
func (pp *PodProcessor) AreAllDependeesPresent(pod Pod) bool {
	return pp.AreAllRelatedPodsPresent(pod, "dependson")
}

func (pp *PodProcessor) IsPodSpecComplete(pod Pod) bool {
	return pp.AreAllDependersPresent(pod) && pp.AreAllDependeesPresent(pod)
}

// Build a directed pod dependency graph
func (pp *PodProcessor) GetPodDependencyGraph(podList []Pod) map[string]map[string]bool {
	podGraph := make(map[string]map[string]bool, 0)

	for _, pod := range podList {
		annotations := pod.Metadata.Annotations
		for k, _ := range annotations {
			vals := strings.Split(k, ".")
			if len(vals) < 2 {
				logger(fmt.Sprintf("ERROR: Incorrect annotation format for pod dependency %s", k))
			}
			relationship, podName := vals[0], vals[1]
			if relationship != "dependson" {
				continue
			}
			_, exists := podGraph[pod.Metadata.Name]
			if !exists {
				podGraph[pod.Metadata.Name] = make(map[string]bool, 0)
			}
			podGraph[pod.Metadata.Name][podName] = true
		}

	}
	return podGraph
}

// Build an undirected graph of pod dependencies from all unscheduled pods
// A pod is added to the graph iff all its dependencies are met, and all the pods that are dependent on this pod are also in the list of unscheduled pods
func (pp *PodProcessor) GetPodGraph() (map[string]map[string]bool, []string) {
	pp.podLock.Lock()
	podList := pp.unscheduledPods
	pp.podLock.Unlock()
	podGraph := make(map[string]map[string]bool, 0)

	skippedPods := make([]string, 0)

	for _, pod := range podList {
		if !pp.IsPodSpecComplete(pod) {
			skippedPods = append(skippedPods, pod.Metadata.Name)
		}

		annotations := pod.Metadata.Annotations
		if len(annotations) == 0 {
			_, exists := podGraph[pod.Metadata.Name]
			if !exists {
				podGraph[pod.Metadata.Name] = make(map[string]bool, 0)
			}
			continue
		}
		for k, _ := range annotations {
			vals := strings.Split(k, ".")
			if len(vals) < 2 {
				logger(fmt.Sprintf("ERROR: Incorrect annotation format for pod dependency %s", k))
			}
			rel, podName := vals[0], vals[1]
			if "dependson" != rel && "dependedby" != rel {
				continue
			}
			logger(fmt.Sprintf("pod = %s rel = %s other pod = %s", pod.Metadata.Name, rel, podName))
			_, exists := podGraph[pod.Metadata.Name]
			if !exists {
				podGraph[pod.Metadata.Name] = make(map[string]bool, 0)
			}

			_, exists = podGraph[podName]
			if !exists {
				podGraph[podName] = make(map[string]bool, 0)
			}
			podGraph[pod.Metadata.Name][podName] = true
			podGraph[podName][pod.Metadata.Name] = true
		}

	}
	/*for srcName, deps := range podGraph {
		logger("Pod: " + srcName)
		logger("Dependers: ")
		fmtStr := ""
		for dstName, _ := range deps {
			fmtStr += dstName + ", "

		}
		logger("Dependers: " + fmtStr)

	}*/
	return podGraph, skippedPods
}

func isInList(elem string, list []string) bool {
	for _, e := range list {
		if e == elem {
			return true
		}
	}
	return false
}

// Perform bfs starting from PodName
func (pp *PodProcessor) GetPodGroup(podName string, podGraph map[string]map[string]bool) map[string]map[string]bool {
	visited := make(map[string]bool, 0)
	visited[podName] = true
	queue := make([]string, 0)
	queue = append(queue, podName)
	for {
		if len(queue) == 0 || queue == nil {
			break
		}
		//logger(fmt.Sprintf("q has %d pods", len(queue)))
		pod := queue[0]
		if len(queue) > 1 {

			queue = queue[1:len(queue)]
		} else {
			queue = nil
		}

		_, exists := podGraph[pod]
		if !exists {
			panic("Unknown pod " + pod)
		}
		visited[pod] = true
		for node, _ := range podGraph[pod] {
			_, visited := visited[node]
			if !visited && !isInList(node, queue) {
				queue = append(queue, node)
			}
		}
	}
	//podList := make([]string, 0)
	podSubgraph := make(map[string]map[string]bool, 0)
	for pod, v := range visited {
		if v == true {
			//podList = append(podList, pod)
			podSubgraph[pod] = make(map[string]bool, 0)
			for neighbor, _ := range podGraph[pod] {
				podSubgraph[pod][neighbor] = true
			}

		}
	}
	return podSubgraph
}

// pod group is a set of pods that need to be scheduled together. They represent an application spec
// We perform an undirected graph traversal and keep track of the connected components in the graph
func (pp *PodProcessor) GetPodGroups(podGraph map[string]map[string]bool, skippedPods []string) []map[string]map[string]bool {
	visited := make(map[string]bool, 0)
	podGroups := make([]map[string]map[string]bool, 0)
	for {
		if len(visited) == len(podGraph) {
			break
		}
		for pod, _ := range podGraph {
			_, exists := visited[pod]
			if exists {
				continue
			}
			podSubgraph := pp.GetPodGroup(pod, podGraph)

			logger(fmt.Sprintf("Got %d pods from %s ", len(podSubgraph), pod))
			skip := false
			for p, _ := range podSubgraph {
				if isInList(p, skippedPods) {
					logger(fmt.Sprintf("Pod %s was skipped, will exclude this pod group", p))
					skip = true
					break
				}
			}

			for p, _ := range podSubgraph {
				visited[p] = true
			}
			if !skip {
				podGroups = append(podGroups, podSubgraph)
			}
		}
	}
	return podGroups
}

func (pp *PodProcessor) MarkScheduled(pods []Pod) {
	pp.podLock.Lock()
	for _, pod := range pods {
		podName := pod.Metadata.Name
		_, exists := pp.unscheduledPods[podName]
		if exists {
			delete(pp.unscheduledPods, podName)
		}
	}
	pp.podLock.Unlock()
}

// return first pod group that is unscheduled
func (pp *PodProcessor) GetUnscheduledPods() (map[string]Pod, map[string]map[string]bool) {
	pp.podLock.Lock()
	podList := pp.unscheduledPods
	pp.podLock.Unlock()

	unscheduled := make(map[string]Pod, 0)
	podGroup := make(map[string]map[string]bool, 0)
	podGraph, skippedPods := pp.GetPodGraph()
	if len(podGraph) == 0 {
		return unscheduled, podGroup
	}
	podGroups := pp.GetPodGroups(podGraph, skippedPods)

	for podName, _ := range podGroups[0] {
		pod, exists := podList[podName]
		if exists {
			unscheduled[podName] = pod
		} else {
			panic("Unknown pod " + podName)
		}
	}
	return unscheduled, podGroups[0]
}
