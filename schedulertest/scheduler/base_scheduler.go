package meshscheduler

import (
    "fmt"
    "github.com/golang/glog"
)

type BaseScheduler struct {
    Nodes               map[string]Node                // nodes indexed by name
    Routes              map[string]map[string]Route    // routes indexed by src and dst
    Assignments         map[string]map[string]string    // app id / component id mapped to node id
    DeploymentStatus    map[string]string               // App id -> deployment status (DEPLOYED/WAITING/COMPLETED)
    Links               map[string]map[string]*LinkBandwidth  // src -> dst -> link bw
}




func (opt *BaseScheduler) InitScheduler(nodes map[string]Node, routes map[string]map[string]Route, links map[string]map[string]*LinkBandwidth) {
    opt.ResetState( nodes, routes, links)
    for src, dstPath := range opt.Routes {
        for dst, path := range dstPath{
            bbw, _ := path.FindBottleneckBw()
            path.BwInUse = 0
            path.BwCapacity = bbw
            opt.Routes[src][dst] = path
        }
    }
    opt.Assignments = make(map[string]map[string]string, 0)
}


func (opt *BaseScheduler) LogAssignments() {
    glog.Info("\nAppId,ComponentId,NodeId\n")
    for app, comps := range opt.Assignments {
        for comp, nodeId := range comps {
            glog.Infof("%s,%s,%s\n", app, comp, nodeId)
        }
    } 
}
func (opt *BaseScheduler) PrintAssignments() {
    fmt.Println("\nAppId,ComponentId,NodeId\n")
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
    glog.Infof("\nSrc,Dst,BwCapacity,BwInUse\n")
    for _, dstPath := range opt.Routes {
        for _, path := range dstPath {
            glog.Infof("%s,%s,%d,%d\n",path.Src, path.Dst, path.BwCapacity, path.BwInUse)
        }
    }
}
func (opt *BaseScheduler) PrintState() {
    fmt.Printf("\nNodeId,CPUCapacity,CPUInUse,MemoryCapacity,MemoryInUse\n")
    for nodeId, n := range opt.Nodes {
        fmt.Printf("%s,%d,%d,%d,%d\n", nodeId, n.CpuCapacity, n.CpuInUse, n.MemoryCapacity, n.MemoryInUse)
    }
    fmt.Printf("\nSrc,Dst,BwCapacity,BwInUse\n")
    for _, dstPath := range opt.Routes {
        for _, path := range dstPath {
            fmt.Printf("%s,%s,%d,%d\n",path.Src, path.Dst, path.BwCapacity, path.BwInUse)
        }
    }
}

func (opt *BaseScheduler) VerifyFit(assignment map[string]map[string]string, app Application, comp Component) (bool, error) {
    appDeployment, exists := assignment[app.AppId]
    
    if !exists {
        return false, &NotFoundError{Msg: "app " + app.AppId + " not found"}
    }
    nodeId, _ := appDeployment[comp.ComponentId]
    nodeState, exists := opt.Nodes[nodeId]
    if !exists {
        return false, &NotFoundError{Msg: "node " + nodeId + " not found"}
    }

    if nodeState.CpuInUse + comp.Cpu > nodeState.CpuCapacity {
        return false, &InsufficientResourceError{ResourceType:"CPU", NodeId:nodeId}
    } else if nodeState.MemoryInUse + comp.Memory > nodeState.MemoryCapacity {
        return false, &InsufficientResourceError{ResourceType:"Memory", NodeId:nodeId}
    }
    for dependency, bw := range comp.Bandwidth {
        depNode, exists := appDeployment[dependency]
        if exists {
            path, exists := opt.Routes[nodeId][depNode]
            if !exists {
                return false, &InsufficientResourceError{ResourceType:"PathBandwidth", NodeId: nodeId +":" + depNode}
            }
            glog.Infof("src node = %s dep node = %s bw available = %d bw in use = %d bw needed = %d\n", nodeId, depNode, path.BwCapacity, path.BwInUse, bw)
            if path.BwInUse + bw > path.BwCapacity {
                return false, &InsufficientResourceError{ResourceType:"PathBandwidth", NodeId:nodeId + ":" + depNode}
            }
        }
    }

    return true, nil
}

func (opt *BaseScheduler) CopyState() (map[string]Node, map[string]map[string]Route, map[string]map[string]*LinkBandwidth){
    oldState := make(map[string]Node, 0)
    for nodeId, state := range opt.Nodes{
        oldState[nodeId] = state
    }
    oldLinks := make(map[string]map[string]*LinkBandwidth, 0)
    for src, dstLink := range opt.Links{
        curDstLink := make(map[string]*LinkBandwidth, 0)
        for dst, link := range dstLink {

            curDstLink[dst] = &LinkBandwidth{Src:link.Src, Dst:link.Dst, BwCapacity:link.BwCapacity, BwInUse:link.BwInUse}

        }
        oldLinks[src] = curDstLink
    }
    oldRoutes := make(map[string]map[string]Route, 0)
    for src, dstRoute := range opt.Routes{
        curDstRoute := make(map[string]Route, 0)
        for dst, route := range dstRoute {

            curDstRoute[dst] = route
            for idx, pbw := range curDstRoute[dst].PathBw{
                oldLink, _ := oldLinks[pbw.Src][pbw.Dst]
                curDstRoute[dst].PathBw[idx] = oldLink
            }

        }
        oldRoutes[src] = curDstRoute
    }
    return oldState, oldRoutes, oldLinks
}

func (opt *BaseScheduler) ResetState(nodes map[string]Node, routes map[string]map[string]Route, links map[string]map[string]*LinkBandwidth) {
    opt.Nodes = nodes
    opt.Links = links
    opt.Routes = make(map[string]map[string]Route, 0)
    for src, dstRoute := range routes{
        _, exists := opt.Routes[src]
        if !exists {
            opt.Routes[src] = make(map[string]Route, 0)
        }
        for dst, route := range dstRoute {
            //fmt.Printf("src = %s dst = %s\n", src,dst)
            opt.Routes[src][dst] = route
            for idx, pbw := range opt.Routes[src][dst].PathBw {
                oldLink, _ := links[pbw.Src][pbw.Dst]
                opt.Routes[src][dst].PathBw[idx] = oldLink
            }

        }
    }
 
}

func (opt *BaseScheduler) UpdatePaths(bottleneckLink *LinkBandwidth) {
    for src, dstMap := range opt.Routes {
        for dst, path :=  range dstMap {
            path.RecomputeBw(bottleneckLink)
            opt.Routes[src][dst] = path
        }
    }    
}



func (opt *BaseScheduler) Schedule(app Application) {
}

