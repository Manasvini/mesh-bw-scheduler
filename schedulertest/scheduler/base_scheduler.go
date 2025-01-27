package meshscheduler

import (
	"fmt"
	"github.com/golang/glog"
    "sort"
)
const MaxUint = ^uint(0) 

type BaseScheduler struct {
	Nodes            NodeMap
	Routes           RouteMap
	Assignments      AppCompAssignment
	DeploymentStatus DeploymentStateMap
	Links            LinkMap
}

func (opt *BaseScheduler) InitScheduler(nodes NodeMap, routes RouteMap, links LinkMap) {
	opt.ResetState(nodes, routes, links)
	for src, dstPath := range opt.Routes {
		for dst, path := range dstPath {
			bbw, _ := path.FindBottleneckBw()
			path.BwInUse = 0
			path.BwCapacity = bbw
			opt.Routes[src][dst] = path
		}
	}
	opt.Assignments = make(AppCompAssignment, 0)
}

func (opt *BaseScheduler) LogAssignmentsHelper(assignment AppCompAssignment) {
	glog.Info("\nAppId,ComponentId,NodeId\n")
	for app, comps := range assignment {
		for comp, nodeId := range comps {
			glog.Infof("%s,%s,%s\n", app, comp, nodeId)
		}
	}
}
func (opt *BaseScheduler) LogAssignments() {
    opt.LogAssignmentsHelper(opt.Assignments)
}

func (opt *BaseScheduler) PrintAssignments() {
	fmt.Println("\nAppId,ComponentId,NodeId")
	for app, comps := range opt.Assignments {
		for comp, nodeId := range comps {
			fmt.Printf("%s,%s,%s\n", app, comp, nodeId)
		}
	}
}

func (opt *BaseScheduler) LogState() {
	glog.Infof("\nNodeId,CPUCapacity,CPUInUse,MemoryCapacity,MemoryInUse\n")
	for nodeId, n := range opt.Nodes {
		glog.Infof("%s,%d,%d,%d,%d\n", nodeId, n.CpuCapacity, n.CpuInUse, n.MemoryCapacity, n.MemoryInUse)
	}
	glog.Infof("Links\nSrc,Dst,BwCapacity,BwInUse\n")
	for _, dstLink := range opt.Links {
		for _, link := range dstLink {
			glog.Infof("%s,%s,%f,%f\n", (*link).Src, (*link).Dst, (*link).BwCapacity, (*link).BwInUse)
		}
	}
	glog.Infof("Paths\nSrc,Dst,BwCapacity,BwInUse\n")
	for _, dstPath := range opt.Routes {
		for _, path := range dstPath {
			glog.Infof("%s,%s,%f,%f\n", path.Src, path.Dst, path.BwCapacity, path.BwInUse)
		}
	}
}
func (opt *BaseScheduler) PrintState() {
	fmt.Printf("\nNodeId,CPUCapacity,CPUInUse,MemoryCapacity,MemoryInUse\n")
	for nodeId, n := range opt.Nodes {
		fmt.Printf("%s,%d,%d,%d,%d\n", nodeId, n.CpuCapacity, n.CpuInUse, n.MemoryCapacity, n.MemoryInUse)
	}
	fmt.Printf("Links\nSrc,Dst,BwCapacity,BwInUse\n")
	for _, dstLink := range opt.Links {
		for _, link := range dstLink {
			fmt.Printf("%s,%s,%f,%f\n", (*link).Src, (*link).Dst, (*link).BwCapacity, (*link).BwInUse)
		}
	}
	fmt.Printf("Paths\nSrc,Dst,BwCapacity,BwInUse\n")
	for _, dstPath := range opt.Routes {
		for _, path := range dstPath {
			fmt.Printf("%s,%s,%f,%f\n", path.Src, path.Dst, path.BwCapacity, path.BwInUse)
		}
	}
}

func (opt *BaseScheduler) VerifyFit(assignment AppCompAssignment, app Application, comp Component) (bool, error) {
	appDeployment, exists := assignment[app.AppId]

	if !exists {
		return false, &NotFoundError{Msg: "app " + app.AppId + " not found"}
	}
	nodeId, _ := appDeployment[comp.ComponentId]
	nodeState, exists := opt.Nodes[nodeId]
	if !exists {
		return false, &NotFoundError{Msg: "node " + nodeId + " not found"}
	}

	if nodeState.CpuInUse+comp.Cpu > nodeState.CpuCapacity {
		return false, &InsufficientResourceError{ResourceType: "CPU", NodeId: nodeId}
	} 

    if nodeState.MemoryInUse+comp.Memory > nodeState.MemoryCapacity {
		return false, &InsufficientResourceError{ResourceType: "Memory", NodeId: nodeId}
	}
	for dependency, bw := range comp.Bandwidth {
		depNode, exists := appDeployment[dependency]
		if exists {
			path, exists := opt.Routes[nodeId][depNode]
			if !exists {
				return false, &InsufficientResourceError{ResourceType: "PathBandwidth", NodeId: nodeId + ":" + depNode}
			}
			//glog.Infof("src node = %s dep node = %s bw available = %d bw in use = %d bw needed = %d\n", nodeId, depNode, path.BwCapacity, path.BwInUse, bw)
			if path.BwInUse+bw > path.BwCapacity {
				return false, &InsufficientResourceError{ResourceType: "PathBandwidth", NodeId: nodeId + ":" + depNode}
			}
		}
	}

	return true, nil
}
func (opt *BaseScheduler) CopyNodes(nmap NodeMap) NodeMap {
    oldState := make(NodeMap, 0)
	for nodeId, state := range nmap {
		oldState[nodeId] = Node{NodeId:state.NodeId, CpuCapacity:state.CpuCapacity, CpuInUse:state.CpuInUse, MemoryCapacity:state.MemoryCapacity, MemoryInUse:state.MemoryInUse}
	}
    return oldState
}

func (opt *BaseScheduler) CopyRoutes(rmap RouteMap, lmap LinkMap) (RouteMap, LinkMap) {
    oldLinks := make(LinkMap, 0)
	for src, dstLink := range lmap {
		_, exists := oldLinks[src]
		if !exists {
			oldLinks[src] = make(map[string]*LinkBandwidth, 0)
		}
		for dst, link := range dstLink {

			oldLinks[src][dst] = &LinkBandwidth{Src: (*link).Src, Dst: (*link).Dst, BwCapacity: (*link).BwCapacity, BwInUse: (*link).BwInUse}

		}
	}
	oldRoutes := make(RouteMap, 0)
	for src, dstRoute := range rmap {
		_, exists := oldRoutes[src]
		if !exists {
			oldRoutes[src] = make(map[string]Route, 0)
		}
		for dst, route := range dstRoute {

			pathBw := make([]*LinkBandwidth, 0)
			for _, pbw := range rmap[src][dst].PathBw {
				oldLink, _ := oldLinks[pbw.Src][pbw.Dst]
				pathBw = append(pathBw, oldLink)
			}
			oldRoutes[src][dst] = Route{Src: route.Src, Dst: route.Dst, BwCapacity: route.BwCapacity, BwInUse: route.BwInUse, PathBw: pathBw}

		}
		//oldRoutes[src] = curDstRoute
	}
	return  oldRoutes, oldLinks

}

func deepCopy(assignments AppCompAssignment) AppCompAssignment{
    newAssignment := make(AppCompAssignment, 0)
    for app, assignment := range assignments{
        newAssignment[app] = make(map[string]string, 0)
        for comp, node := range assignment{
            newAssignment[app][comp] = node
        }
    }
    return newAssignment
}

func (opt *BaseScheduler) CopyState() (NodeMap, RouteMap, LinkMap) {
	oldState := make(NodeMap, 0)
	for nodeId, state := range opt.Nodes {
		oldState[nodeId] = state
	}
	oldLinks := make(LinkMap, 0)
	for src, dstLink := range opt.Links {
		_, exists := oldLinks[src]
		if !exists {
			oldLinks[src] = make(map[string]*LinkBandwidth, 0)
		}
		for dst, link := range dstLink {

			oldLinks[src][dst] = &LinkBandwidth{Src: (*link).Src, Dst: (*link).Dst, BwCapacity: (*link).BwCapacity, BwInUse: (*link).BwInUse}

		}
	}
	oldRoutes := make(RouteMap, 0)
	for src, dstRoute := range opt.Routes {
		_, exists := oldRoutes[src]
		if !exists {
			oldRoutes[src] = make(map[string]Route, 0)
		}
		for dst, route := range dstRoute {

			pathBw := make([]*LinkBandwidth, 0)
			for _, pbw := range opt.Routes[src][dst].PathBw {
				oldLink, _ := oldLinks[pbw.Src][pbw.Dst]
				pathBw = append(pathBw, oldLink)
			}
			oldRoutes[src][dst] = Route{Src: route.Src, Dst: route.Dst, BwCapacity: route.BwCapacity, BwInUse: route.BwInUse, PathBw: pathBw}

		}
		//oldRoutes[src] = curDstRoute
	}
	return oldState, oldRoutes, oldLinks
}

func (opt *BaseScheduler) ResetState(nodes NodeMap, routes RouteMap, links LinkMap) {
	opt.Nodes = nodes
	opt.Links = make(LinkMap, 0)
	opt.Routes = make(RouteMap, 0)
	for src, dstLink := range links {
		_, exists := opt.Links[src]
		if !exists {
			opt.Links[src] = make(map[string]*LinkBandwidth, 0)
		}
		for dst, link := range dstLink {
			opt.Links[src][dst] = &LinkBandwidth{Src: (*link).Src, Dst: (*link).Dst, BwCapacity: (*link).BwCapacity, BwInUse: (*link).BwInUse}
		}
	}
	for src, dstRoute := range routes {
		_, exists := opt.Routes[src]
		if !exists {
			opt.Routes[src] = make(map[string]Route, 0)
		}
		for dst, route := range dstRoute {
			opt.Routes[src][dst] = route
			for idx, pbw := range opt.Routes[src][dst].PathBw {
				oldLink, _ := opt.Links[pbw.Src][pbw.Dst]
				route.PathBw[idx] = oldLink

			}
			_, link := route.FindBottleneckBw()
			route.RecomputeBw(link)
			opt.Routes[src][dst] = route
		}
	}

}

func (opt *BaseScheduler) UpdatePaths(links LinkMap, routes RouteMap) {
	for src, dstMap := range routes {
		for dst, path := range dstMap {
			_, blink := path.FindBottleneckBw()
			path.RecomputeBw(blink)
			routes[src][dst] = path
			for _, link := range path.PathBw {
				optLink, exists := links[link.Src][link.Dst]
				if exists {
					if optLink.BwInUse != link.BwInUse {
						optLink.BwInUse = link.BwInUse
					}
				}
			}
		}
	}

}

type CompTotalBw struct{
    compId string
    bw      float64
    degree int
}
func (opt *BaseScheduler) GetCompOrder(comps map[string]Component)[]CompTotalBw{
    compTotalBw := make([]CompTotalBw, 0)
    for compId, comp := range comps {
        bwSum := 0.0
        for _, bw := range comp.Bandwidth {
           bwSum += bw
        }
    
        compTotalBw = append(compTotalBw, CompTotalBw{compId:compId, bw: bwSum, degree:len(comp.Bandwidth)})
    }
    sort.Slice(compTotalBw, func(i int, j int) bool{
        return compTotalBw[i].bw >= compTotalBw[j].bw
    })
    return compTotalBw
}

type NodeTotalBw struct{
        nodeId string
        bw      float64
        degree int
}
func (opt *BaseScheduler) GetNodeOrder(nodes NodeMap, links LinkMap)[]NodeTotalBw{
    nodeTotalBw := make([]NodeTotalBw, 0)
    for node, _ := range nodes {
        bwSum := 0.0
        for dst, link := range links[node] {
            if dst == node{
                continue
            }
            bwSum += link.BwCapacity 
        }
        //glog.Infof("node id = %s bw = %f\n", node, bwSum)
        nodeTotalBw = append(nodeTotalBw, NodeTotalBw{nodeId:node, bw: bwSum, degree:len(links[node])})
    }
    sort.Slice(nodeTotalBw, func(i int, j int) bool{
        if nodeTotalBw[i].bw >= nodeTotalBw[j].bw {
            return true
        }
        return false
    })
    return nodeTotalBw
}
func (opt *BaseScheduler) Schedule(app Application) {
}
