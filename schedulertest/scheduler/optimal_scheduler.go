package meshscheduler

import (
    "fmt"
    "github.com/golang/glog"
    "strconv"
)

type OptimalScheduler struct {
    //Nodes               map[string]Node                // nodes indexed by name
    //Routes              map[string]map[string]Route    // routes indexed by src and dst
    //Assignments         map[string]map[string]string    // app id / component id mapped to node id
    //DeploymentStatus    map[string]string               // App id -> deployment status (DEPLOYED/WAITING/COMPLETED)
    //Links               map[string]map[string]*LinkBandwidth  // src -> dst -> link bw
    BaseScheduler
}


/*func (opt *OptimalScheduler) InitScheduler(nodes map[string]Node, routes map[string]map[string]Route, links map[string]map[string]*LinkBandwidth) {
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
*/
func NewOptimalScheduler()(*OptimalScheduler) {
    return &OptimalScheduler{}
}

/*func (opt *OptimalScheduler) LogAssignments() {
    glog.Info("\nAppId,ComponentId,NodeId\n")
    for app, comps := range opt.Assignments {
        for comp, nodeId := range comps {
            glog.Infof("%s,%s,%s\n", app, comp, nodeId)
        }
    } 
}
func (opt *OptimalScheduler) PrintAssignments() {
    fmt.Println("\nAppId,ComponentId,NodeId\n")
    for app, comps := range opt.Assignments {
        for comp, nodeId := range comps {
            fmt.Printf("%s,%s,%s\n", app, comp, nodeId)
        }
    } 
}

func (opt *OptimalScheduler) LogState() {
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

func (opt *OptimalScheduler) VerifyFit(assignment map[string]map[string]string, app Application, comp Component) (bool, error) {
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
            //fmt.Printf("src = %s dst = %s\n", src,dst)
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
*/
func (opt *OptimalScheduler) MakeAssignment(app Application, component Component, currentAssignment map[string]map[string]string, nodeId string) (error,  map[string]map[string]string){
    currentAssignment[app.AppId][component.ComponentId] = nodeId
    possible, err := opt.VerifyFit(currentAssignment, app, component)
    oldState, oldRoutes, oldLinks := opt.CopyState()
    glog.Infof("Test comp %s on node %s\n", component.ComponentId, nodeId)
    if possible == true {
        n := opt.Nodes[nodeId] 
        n.CpuInUse += component.Cpu
        n.MemoryInUse += component.Memory
        opt.Nodes[nodeId] = n
        // can this node accommodate all bw contraints of this component to the existing component
        for dependency, bw := range component.Bandwidth {
            depNode, exists := currentAssignment[app.AppId][dependency]
            if exists {
               // n.BandwidthInUse[depNode] += bw
                path, exists := opt.Routes[nodeId][depNode]
                if !exists {
                    delete(currentAssignment[app.AppId], component.ComponentId)
                    opt.ResetState(oldState, oldRoutes, oldLinks)
                    return &InsufficientResourceError{ResourceType:"PathBandwidth", NodeId:depNode  +":" + nodeId}, currentAssignment
                }
                bottleneckBw, bottleneckLink := path.FindBottleneckBw()
                glog.Infof("check assignment for %s to %s  with dep on %s bw = %d available = %d\n", component.ComponentId, nodeId, depNode, bw, bottleneckBw) 
                glog.Infof("bottleneck link is %s to %s\n", bottleneckLink.Src, bottleneckLink.Dst)
                if bottleneckBw - bw > 0 {
                    bottleneckLink.BwInUse += bw
                    path.SetPathBw(bottleneckLink.BwInUse)
                    opt.UpdatePaths(bottleneckLink)

                    reverseLink, _ := opt.Links[bottleneckLink.Dst][bottleneckLink.Src]
                    reverseLink.BwInUse += bw
                    reversePath, _ := opt.Routes[bottleneckLink.Dst][bottleneckLink.Src]
                    reversePath.SetPathBw(reverseLink.BwInUse)
                    opt.UpdatePaths(reverseLink)

                } else {
                    delete(currentAssignment[app.AppId], component.ComponentId)
                    opt.ResetState(oldState, oldRoutes, oldLinks)

                    return &InsufficientResourceError{ResourceType:"PathBandwidth", NodeId:nodeId}, currentAssignment
                }
            }

        }
        // can all the already scheduled dependencies to this component be satisfied by this node
        appAssignment := currentAssignment[app.AppId]
        for compId, nId := range appAssignment {
            compDeps, _ := app.Components[compId]
            for dep, _ := range compDeps.Bandwidth{
                depNode, exists := appAssignment[dep]
                if !exists {
                    continue
                }
                if dep == component.ComponentId {
                    
                    path, exists := opt.Routes[nId][nodeId]
                    glog.Infof("src = %s dst = %s\n", nId, nodeId)
                    if !exists {
                        delete(currentAssignment[app.AppId], component.ComponentId)
                        opt.ResetState(oldState, oldRoutes, oldLinks)
                        return &InsufficientResourceError{ResourceType:"PathBandwidth", NodeId:depNode  +":" + nodeId}, currentAssignment
                    }
                    bottleneckBw, bottleneckLink := path.FindBottleneckBw()
                    if bottleneckBw - app.Components[compId].Bandwidth[dep] < 0{
                        delete(currentAssignment[app.AppId], component.ComponentId)
                        opt.ResetState(oldState, oldRoutes, oldLinks)

                        return &InsufficientResourceError{ResourceType:"PathBandwidth", NodeId:depNode  +":" + nodeId}, currentAssignment
 
                    }
                    glog.Infof("check bw for comps %s to %s  on nodes %s and %s\n", compId, component.ComponentId, nId,nodeId ) 
                    glog.Infof("bottleneck link is %s to %s bw = %d\n", bottleneckLink.Src, bottleneckLink.Dst, bottleneckBw)
                    bottleneckLink.BwInUse += app.Components[compId].Bandwidth[dep]
                    path.SetPathBw(bottleneckLink.BwInUse)
                    opt.UpdatePaths(bottleneckLink)

                    reverseLink, _ := opt.Links[bottleneckLink.Dst][bottleneckLink.Src]
                    reverseLink.BwInUse += app.Components[compId].Bandwidth[dep]
                    reversePath, _ := opt.Routes[bottleneckLink.Dst][bottleneckLink.Src]
                    reversePath.SetPathBw(reverseLink.BwInUse)
                    opt.UpdatePaths(reverseLink)
                }

            }
        }
        opt.Nodes[nodeId] = n
        return nil, currentAssignment
    }
    return err, currentAssignment
}

func (opt *OptimalScheduler) SchedulerHelper(app Application, currentAssignment map[string]map[string]string) bool{
    glog.Info("current assignment\n")
    appAssignment, _ :=  currentAssignment[app.AppId]
    for comp, nodeId := range appAssignment{
        glog.Infof("comp = %s node = %s\n", comp, nodeId)
    }
    if len(appAssignment) == len(app.Components){
            opt.Assignments[app.AppId] = make(map[string]string, 0)
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
            opt.LogState()
            opt.LogAssignments()
            glog.Infof("processing comp %s on node %s \n", comp.ComponentId, nodeId)
            err, assignment := opt.MakeAssignment(app, comp, currentAssignment, nodeId)
            if err != nil {
                glog.Info(err)
            }
            if err == nil {
                currentAssignment = assignment
            } else {
                continue
            }
            glog.Info("After assignment, state is")
            opt.LogState()
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
    currentAssignment[app.AppId] = make(map[string]string, 0)
    oldState, oldRoutes, oldLinks := opt.CopyState()
    possible := opt.SchedulerHelper(app, currentAssignment)
    fmt.Printf("is possible for app %s to be scheduled = %d\n", app.AppId, strconv.FormatBool(possible))
    if !possible {
        opt.ResetState(oldState, oldRoutes, oldLinks)
    } 
    opt.PrintState()
    opt.PrintAssignments()
}

