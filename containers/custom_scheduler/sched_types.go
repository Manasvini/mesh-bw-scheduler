package main

import "sort"

type Resource struct {
	cpu    int64
	memory int64
	name   string
}

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

type KubeClientIntf interface {
	GetNodes() (*NodeList, error)
	WatchUnscheduledPods() (<-chan Pod, <-chan error)
	WaitForProxy() int
	GetNodeMetrics() (*NodeMetricsList, error)
	GetUnscheduledPods() ([]*Pod, error)
	GetPods() ([]*PodList, error)
	Bind(pod Pod, node Node) error
}
