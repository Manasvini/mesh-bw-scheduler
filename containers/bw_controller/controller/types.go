package bw_controller

type Pod struct {
	podId        string
	deployedNode string
	podName      string
}

type PodDependency struct {
	source      string
	destination string
	latency     float64
	bandwidth   float64
}

type PodSet map[string]Pod
type PodDeps map[string]map[string]PodDependency

type Link struct {
	source      string
	destination string
	latency     float64
	bandwidth   float64
}

type LinkSet map[string]map[string]Link

type Path struct {
	source      string
	destination string
	hops        []string
	bandwidth   float64
	latency     float64
}
type PathSet map[string]map[string]Path
