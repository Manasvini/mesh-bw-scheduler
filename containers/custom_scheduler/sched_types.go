package main

import "sort"

type Resource struct {
	cpu    int64
	memory int64
	name   string
}

const RESOURCE_DIFF_THRESHOLD float64 = 0.25
type Resources []Resource

func (resources Resources) Len() int {
	return len(resources)
}

func (resources Resources) Swap(i, j int) {
	resources[i], resources[j] = resources[j], resources[i]
}

func (resources Resources) Less(i, j int) bool {
	if resources[i].cpu >= resources[j].cpu {
		return true
	}
	return resources[i].memory > resources[j].memory
}

func sortNodes(resources []Resource) {
	sort.Sort(Resources(resources))

}

func getResourceByNodeName( resources []Resource, nodeName string) (Resource, int){
	res := Resource{name:""}
	for idx, r := range resources {
		if r.name == nodeName {
			return r, idx
		}
	}
	return res, -1

}

type NodeResourceWithDeps struct{
	resource	Resource
	numDeps		int
}
type NodeResourceDepsList []NodeResourceWithDeps

func (nodeResDeps NodeResourceDepsList) Len() int {
	return len(nodeResDeps)
}

func (nodeResDeps NodeResourceDepsList) Swap(i, j int){
	nodeResDeps[i], nodeResDeps[j] = nodeResDeps[j], nodeResDeps[i]
}

func (nodeResDeps NodeResourceDepsList) Less(i, j int) bool {
	if nodeResDeps[i].numDeps > nodeResDeps[j].numDeps {
		if float64(nodeResDeps[i].resource.cpu) >= RESOURCE_DIFF_THRESHOLD * float64(nodeResDeps[j].resource.cpu) &&  float64(nodeResDeps[i].resource.memory) >= RESOURCE_DIFF_THRESHOLD * float64(nodeResDeps[j].resource.memory){ 
			return true
		}
		return false
	}
	return nodeResDeps[i].resource.cpu >= nodeResDeps[j].resource.cpu || nodeResDeps[i].resource.memory > nodeResDeps[j].resource.memory
}

func sortNodesWithDeps(nodeResWithDeps []NodeResourceWithDeps) {
	sort.Sort(NodeResourceDepsList(nodeResWithDeps))
}

type KubeClientIntf interface {
	GetNodes() (*NodeList, error)
	WatchUnscheduledPods() (<-chan Pod, <-chan error)
	WaitForProxy() int
	GetNodeMetrics() (*NodeMetricsList, error)
	GetUnscheduledPods() ([]*Pod, error)
	GetPods() ([]*PodList, error)
	Bind(pod Pod, node Node) error
}

