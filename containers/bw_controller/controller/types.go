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

type Traffic struct {
	source      string
	destination string
	bytes       float64
}

type TrafficSet map[string]map[string]Traffic

type Pair struct {
	Key   string
	Value float64
}

type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PairList) Less(i, j int) bool { return p[i].Value > p[j].Value }
