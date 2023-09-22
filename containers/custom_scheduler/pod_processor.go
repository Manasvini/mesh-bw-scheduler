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
	client          KubeClientIntf
}

func NewPodProcessor(kcl KubeClientIntf) *PodProcessor {
	mu := &sync.Mutex{}
	unscheduledPods := make(map[string]Pod, 0)
	pp := &PodProcessor{unscheduledPods: unscheduledPods, podLock: mu, client: kcl}
	logger("Created pod processor")
	return pp
}
func getPodName(podId string) string {
	pName := ""
	vals := strings.Split(podId, "-")
	for i := 0; i < len(vals)-2; i++ {
		pName += vals[i]
		if i < len(vals)-3 {
			pName += "-"
		}

	}
	return pName
}
func (pp *PodProcessor) AddPod(pod Pod) {
	pp.podLock.Lock()
	pName := pod.Metadata.Name //getPodName(pod.Metadata.Name)

	pp.unscheduledPods[pName] = pod

	pp.podLock.Unlock()
	logger("Added pod " + pName + " to unscheduled pods")
}

func (pp *PodProcessor) IsPodInList(podList []*PodList, podName string) bool {
	for _, pList := range podList {
		for _, pod := range pList.Items {
			pName := getPodName(pod.Metadata.Name)
			if pName == podName && (pod.Status.Phase == "Running" || pod.Status.Phase == "ContainerCreating" || strings.Contains(pod.Status.Phase, "Init")) {
				return true
			} else if pName == podName && (pod.Status.Phase == "ContainerStatusUnknown" || pod.Status.Phase == "Completed" || pod.Status.Phase == "Terminating") {
				continue
			}
		}

	}
	return false

}
func (pp *PodProcessor) AreAllRelatedPodsPresent(pod Pod, relationship string) bool {
	// Dependson: for a pod, check if all  the pods that THIS pod depends on are present
	// Dependedby: for a pod, check if all pods that depend on THIS pod are present
	pp.podLock.Lock()
	podList := pp.unscheduledPods
	pp.podLock.Unlock()
	annotations := pod.Metadata.Annotations
	// format: dependedby.PODNAME
	allPods, err := pp.client.GetPods()
	if err != nil {
		logger(fmt.Sprintf("Got error: %v", err))
	}
	//logger(fmt.Sprintf("find %s for pod %s", relationship, pod.Metadata.Name))
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
			pod := getPodWithName(podName, podList)
			isPodPresent := true
			//logger("pd meta name is " + getPodName(pod.Metadata.Name) + " pd name is" + podName)
			if getPodName(pod.Metadata.Name) != podName {
				isPodPresent = false
			}
			podAlreadyScheduled := pp.IsPodInList(allPods, podName)
			if !isPodPresent && !podAlreadyScheduled {
				logger("Pod " + podName + " not found")
				return false
			}
		}
	}
	//logger(fmt.Sprintf("POd %s has all %s ", getPodName(pod.Metadata.Name), relationship))
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
			_, exists := podGraph[getPodName(pod.Metadata.Name)]
			if !exists {
				podGraph[getPodName(pod.Metadata.Name)] = make(map[string]bool, 0)
			}
			podGraph[getPodName(pod.Metadata.Name)][podName] = true
			logger("add dep " + getPodName(pod.Metadata.Name) + " <-> " + podName)
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
			logger(fmt.Sprintf("Pod %s was skipped", pod.Metadata.Name))
			skippedPods = append(skippedPods, pod.Metadata.Name)
		}

		annotations := pod.Metadata.Annotations
		annCt := 0
		for k, _ := range annotations {
			if strings.Contains(k, "dependson") || strings.Contains(k, "dependedby") {
				annCt += 1
			}
		}
		if len(annotations) == 0 || annCt == 0 {
			_, exists := podGraph[getPodName(pod.Metadata.Name)]
			if !exists {
				podGraph[getPodName(pod.Metadata.Name)] = make(map[string]bool, 0)
			}
			continue
		}
		for k, _ := range annotations {
			vals := strings.Split(k, ".")
			if len(vals) < 2 {
				logger(fmt.Sprintf("ERROR: Incorrect annotation format for pod dependency %s", k))
			}
			rel, podName := vals[0], vals[1]
			//logger(fmt.Sprintf("pod = %s rel = %s other pod = %s", pod.Metadata.Name, rel, podName))
			if "dependson" != rel && "dependedby" != rel {
				continue
			}
			_, exists := podGraph[getPodName(pod.Metadata.Name)]
			if !exists {
				podGraph[getPodName(pod.Metadata.Name)] = make(map[string]bool, 0)
			}

			_, exists = podGraph[podName]
			if !exists {
				podGraph[podName] = make(map[string]bool, 0)
				//logger("add dep " + podName + " for " + getPodName(pod.Metadata.Name))
			}
			podGraph[getPodName(pod.Metadata.Name)][podName] = true
			podGraph[podName][getPodName(pod.Metadata.Name)] = true
		}

	}
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
		//logger("src pod = " + podName + "cur pod = " + pod)
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
				podInfo := getPodWithName(pod, pp.unscheduledPods)
				//podInfo, _ := pp.unscheduledPods[pod]
				if getPodName(podInfo.Metadata.Name) != pod {
					continue
				}
				for ann, _ := range podInfo.Metadata.Annotations {
					vals := strings.Split(ann, ".")
					if vals[0] == "dependson" && neighbor == vals[1] {
						podSubgraph[pod][neighbor] = true
						//logger(fmt.Sprintf("added %s -> %s", pod, neighbor))
						break
					}
				}
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
	for pod, _ := range podGraph {
		if len(visited) == len(podGraph) {
			break
		}
		_, exists := visited[pod]
		if exists {
			continue
		}
		podSubgraph := pp.GetPodGroup(pod, podGraph)

		//logger(fmt.Sprintf("Got %d pods from %s ", len(podSubgraph), pod))
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
	return podGroups
}

func (pp *PodProcessor) MarkScheduled(pods []Pod) {
	pp.podLock.Lock()
	for _, pod := range pods {
		logger("schedule pod = " + pod.Metadata.Name)
		podName := pod.Metadata.Name // getPodName(pod.Metadata.Name)
		_, exists := pp.unscheduledPods[podName]

		if exists {
			delete(pp.unscheduledPods, podName)
		}
	}
	pp.podLock.Unlock()
}

func getPodWithName(podName string, pods map[string]Pod) Pod {
	var pod Pod
	for p, podInfo := range pods {
		//logger("Pod is " + p)
		if getPodName(p) == podName {
			return podInfo
		}
	}
	return pod

}

// return first pod group that is unscheduled
func (pp *PodProcessor) GetUnscheduledPods() (map[string]Pod, map[string]map[string]bool) {
	pp.podLock.Lock()
	podList := pp.unscheduledPods
	pp.podLock.Unlock()
	logger(fmt.Sprintf("Pod list has %d pods", len(podList)))
	unscheduled := make(map[string]Pod, 0)
	podGroup := make(map[string]map[string]bool, 0)
	podGraph, skippedPods := pp.GetPodGraph()
	logger(fmt.Sprintf("Pod graph has %d pods", len(podGraph)))
	if len(podGraph) == 0 {
		return unscheduled, podGroup
	}
	podGroups := pp.GetPodGroups(podGraph, skippedPods)
	if len(podGroups) == 0 {
		return unscheduled, podGroup

	}
	unknownPods := make([]string, 0)
	for podName, _ := range podGroups[0] {
		//logger("pg pod name " + podName)
		pod := getPodWithName(podName, podList)
		exists := true
		if getPodName(pod.Metadata.Name) != podName {
			exists = false
		}
		//if exists {
		unscheduled[pod.Metadata.Name] = pod

		if !exists {
			logger("Pod " + podName + " not in list of unscheduled pods")
			//unknownPods = append(unknownPods, podName)
		}
	}
	for _, pod := range unknownPods {
		delete(podGroups[0], pod)
	}
	return pp.unscheduledPods, podGroups[0]
}
