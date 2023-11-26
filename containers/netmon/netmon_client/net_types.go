package netmon_client

type Link struct {
	Source      string
	Destination string
	Latency     float64
	Bandwidth   float64
}

type LinkSet map[string]map[string]Link

type Path struct {
	Source      string
	Destination string
	Hops        []string
	Bandwidth   float64
	Latency     float64
}
type PathSet map[string]map[string]Path

type Traffic struct {
	Source      string
	Destination string
	Bytes       float64
}

type TrafficSet map[string]map[string]Traffic

type NetmonClientIntf interface {
	Close()
	GetStats(nodeMap map[string]string) (LinkSet, PathSet, TrafficSet)
}
