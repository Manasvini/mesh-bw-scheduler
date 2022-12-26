package meshscheduler

import (
    "fmt"
)

type OptimalScheduler struct {
    Nodes               map[string]Node                // nodes indexed by name
    Routes              map[string]map[string]Route    // routes indexed by src and dst
    Assignments         map[string]map[string]string    // app id / component id mapped to node id
    DeploymentStatus    map[string]string               // App id -> deployment status (DEPLOYED/WAITING/COMPLETED)
    Links               map[string]map[string]*LinkBandwidth  // src -> dst -> link bw
}


func (opt *OptimalScheduler) InitScheduler(nodes map[string]Node, routes map[string]map[string]Route, links map[string]map[string]*LinkBandwidth) {
    opt.ResetState( nodes, routes, links)
}

func NewOptimalScheduler()(*OptimalScheduler) {
    return &OptimalScheduler{}
}


func (opt *OptimalScheduler) PrintAssignments() {
    fmt.Printf("\nAppId,ComponentId,NodeId\n")
    for app, comps := range opt.Assignments {
        for comp, nodeId := range comps {
            fmt.Printf("%s,%s,%s\n", app, comp, nodeId)
        }
    } 
}

func (opt *OptimalScheduler) PrintState() {
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

func (opt *OptimalScheduler) VerifyFit(assignment map[string]map[string]string, app Application) (bool, error) {
    appDeployment, exists := assignment[app.AppId]
    if !exists {
        return false, &NotFoundError{Msg: "app " + app.AppId + " not found"}
    }
    for compId, nodeId := range appDeployment {
        nodeState, exists := opt.Nodes[nodeId]

        if !exists {
            return false, &NotFoundError{Msg: "node " + nodeId + " not found"}
        }

        comp, _ := app.Components[compId]
        if nodeState.CpuInUse + comp.Cpu > nodeState.CpuCapacity {
            return false, &InsufficientResourceError{ResourceType:"CPU", NodeId:nodeId}
        } else if nodeState.MemoryInUse + comp.Memory > nodeState.MemoryCapacity {
            return false, &InsufficientResourceError{ResourceType:"Memory", NodeId:nodeId}
        }
        for dependency, bw := range comp.Bandwidth {
            depNode, exists := appDeployment[dependency]
            if exists {
                path, _ := opt.Routes[nodeId][depNode]
                if path.BwInUse + bw > path.BwCapacity {
                    return false, &InsufficientResourceError{ResourceType:"PathBandwidth", NodeId:nodeId + ":" + depNode}
                }
            }
        }

    }
    return true, nil
}

func (opt *OptimalScheduler) CopyState() (map[string]Node, map[string]map[string]Route, map[string]map[string]*LinkBandwidth){
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

func (opt *OptimalScheduler) ResetState(nodes map[string]Node, routes map[string]map[string]Route, links map[string]map[string]*LinkBandwidth) {
    opt.Nodes = nodes
    opt.Links = links
    opt.Routes = make(map[string]map[string]Route, 0)
    for src, dstRoute := range routes{
        _, exists := opt.Routes[src]
        if !exists {
            opt.Routes[src] = make(map[string]Route, 0)
        }
        for dst, route := range dstRoute {
            fmt.Printf("src = %s dst = %s\n", src,dst)
            opt.Routes[src][dst] = route
            for idx, pbw := range opt.Routes[src][dst].PathBw {
                oldLink, _ := links[pbw.Src][pbw.Dst]
                opt.Routes[src][dst].PathBw[idx] = oldLink
            }

        }
    }
 
}

func (opt *OptimalScheduler) UpdatePaths(bottleneckLink *LinkBandwidth) {
    for src, dstMap := range opt.Routes {
        for dst, path :=  range dstMap {
            path.RecomputeBw(bottleneckLink)
            opt.Routes[src][dst] = path
        }
    }    
}

func (opt *OptimalScheduler) MakeAssignment(app Application, component Component, currentAssignment map[string]map[string]string, nodeId string) (error,  map[string]map[string]string){
    currentAssignment[app.AppId][component.ComponentId] = nodeId
    possible, err := opt.VerifyFit(currentAssignment, app)
    oldState, oldRoutes, oldLinks := opt.CopyState()
    if possible == true &&  err == nil {
        n := opt.Nodes[nodeId] 
        n.CpuInUse += component.Cpu
        n.MemoryInUse += component.Memory
        opt.Nodes[nodeId] = n
        for dependency, bw := range component.Bandwidth {
            depNode, exists := currentAssignment[app.AppId][dependency]
            if exists {
                n.BandwidthInUse[depNode] += bw
                path, _ := opt.Routes[nodeId][depNode]
                bottleneckBw, bottleneckLink := path.FindBottleneckBw()
                if bottleneckBw - bw > 0 {
                    bottleneckLink.BwInUse += bw
                    path.SetPathBw(bottleneckLink.BwInUse)
                    opt.UpdatePaths(bottleneckLink)

                    reverseLink := opt.Links[bottleneckLink.Dst][bottleneckLink.Src]
                    reverseLink.BwInUse += bw
                    path.SetPathBw(reverseLink.BwInUse)
                    opt.UpdatePaths(reverseLink)



                } else {
                    delete(currentAssignment[app.AppId], component.ComponentId)
                    opt.ResetState(oldState, oldRoutes, oldLinks)

                    return &InsufficientResourceError{ResourceType:"PathBandwidth", NodeId:nodeId}, currentAssignment
                }
            }
        }
        opt.Nodes[nodeId] = n
        return nil, currentAssignment
    }
    return err, currentAssignment
}

func (opt *OptimalScheduler) SchedulerHelper(app Application, currentAssignment map[string]map[string]string) bool{
    if len(currentAssignment[app.AppId]) == len(app.Components){
            opt.Assignments[app.AppId] = currentAssignment[app.AppId]
            return true         
    }   
    for _, comp := range app.Components{
        _, exists := currentAssignment[app.AppId][comp.ComponentId]
        if exists {
            continue
        }
        for nodeId, _ := range opt.Nodes {
            oldState, oldRoutes, oldLinks := opt.CopyState()
            err, assignment := opt.MakeAssignment(app, comp, currentAssignment, nodeId)
            if err == nil {
                currentAssignment = assignment
            } else {
                continue
            }
                
            possible := opt.SchedulerHelper(app, currentAssignment)
            if !possible {
                opt.ResetState(oldState, oldRoutes, oldLinks)
            }
            return possible
        }
    }
    return false
}

func (opt *OptimalScheduler) Schedule(app Application) {
    currentAssignment := make(map[string]map[string]string, 0)
    oldState, oldRoutes, oldLinks := opt.CopyState()
    possible := opt.SchedulerHelper(app, currentAssignment)
    if !possible {
        opt.ResetState(oldState, oldRoutes, oldLinks)
    } 
}

