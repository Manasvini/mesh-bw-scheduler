package meshscheduler

import (
    "fmt"
    "github.com/golang/glog"
    "strconv"
)

type OptimalScheduler struct {
    BaseScheduler
}


func NewOptimalScheduler()(*OptimalScheduler) {
    return &OptimalScheduler{}
}

func (opt *OptimalScheduler) InitScheduler(nodes NodeMap, routes RouteMap, links LinkMap) {
   
    opt.ResetState( nodes, routes, links)
    for src, dstPath := range opt.Routes {
        for dst, path := range dstPath{
            bbw, _ := path.FindBottleneckBw()
            path.BwInUse = 0
            path.BwCapacity = bbw
            opt.Routes[src][dst] = path
            
        }
    }
    opt.Assignments = make(AppCompAssignment, 0)
}

func (opt *OptimalScheduler) MakeAssignment(app Application, component Component, currentAssignment AppCompAssignment, nodeId string) (error,  AppCompAssignment){
    currentAssignment[app.AppId][component.ComponentId] = nodeId
    possible, err := opt.VerifyFit(currentAssignment, app, component)
    oldState, oldRoutes, oldLinks := opt.CopyState()
    for src, dstLink := range oldLinks {
        for dst, link := range dstLink {
            glog.Infof("old link src = %s dst = %s bw in use = %d ol addr link %p addr %p\n", src, dst, link.BwInUse, link, opt.Links[src][dst])
        }
    }
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
                    opt.UpdatePaths()
                    return &InsufficientResourceError{ResourceType:"PathBandwidth", NodeId:depNode  +":" + nodeId}, currentAssignment
                }
                bottleneckBw, bottleneckLink := path.FindBottleneckBw()
                glog.Infof("check assignment for %s to %s  with dep on %s bw = %d available = %d\n", component.ComponentId, nodeId, depNode, bw, bottleneckBw) 
                glog.Infof("bottleneck link is %s to %s\n", bottleneckLink.Src, bottleneckLink.Dst)
                if bottleneckBw  > bw {
                    bottleneckLink.BwInUse += bw
                    path.SetPathBw(bottleneckLink.BwInUse)
                    opt.Routes[nodeId][depNode] = path
                    //opt.UpdatePaths()
                    opt.Links[bottleneckLink.Src][bottleneckLink.Dst] = bottleneckLink
                     
                    glog.Infof("updating reverse path %s to %s usage\n", depNode, nodeId)
                    
                    reversePath, _ := opt.Routes[depNode][nodeId]
                    _, reverseLink:= reversePath.FindBottleneckBw()
                    reverseLink.BwInUse += bw
                    reversePath.SetPathBw(reverseLink.BwInUse)
                    //opt.UpdatePaths()
                    opt.Routes[depNode][nodeId] = reversePath
                    opt.Links[reverseLink.Src][reverseLink.Dst] = reverseLink
                    
                    opt.UpdatePaths()  
                    
                    glog.Infof("rev link %s to %s bw in use =%d\n", reverseLink.Src, reverseLink.Dst,  (*(opt.Links[bottleneckLink.Dst][bottleneckLink.Src])).BwInUse)

                } else {
                    delete(currentAssignment[app.AppId], component.ComponentId)

                    opt.ResetState(oldState, oldRoutes, oldLinks)
                    opt.UpdatePaths()
                    return &InsufficientResourceError{ResourceType:"PathBandwidth", NodeId:nodeId}, currentAssignment
                }
            }

        }
        glog.Infof("Updated %s to all deps", component.ComponentId)
        opt.LogState()
        for src, dstLink := range oldLinks{
            for dst, link := range dstLink {
                glog.Infof("Link src = %s dst = %s bw in use = %d ol addr %p link addr %p \n", src, dst, link.BwInUse, link, opt.Links[src][dst])
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
                    glog.Infof("src = %s dst = %s bw in use =%d\n", nId, nodeId, path.BwInUse)
                    if !exists {
                        glog.Infof("No path from %s to %s, reset staate\n", nId, nodeId)
                        delete(currentAssignment[app.AppId], component.ComponentId)
                        opt.ResetState(oldState, oldRoutes, oldLinks)
                        opt.UpdatePaths()
                        return &InsufficientResourceError{ResourceType:"PathBandwidth", NodeId:depNode  +":" + nodeId}, currentAssignment
                    }
                    depCompBw, _ := app.Components[compId].Bandwidth[dep]
                    bottleneckBw, bottleneckLink := path.FindBottleneckBw()
                    if bottleneckBw - depCompBw < 0{
                        delete(currentAssignment[app.AppId], component.ComponentId)
                        opt.UpdatePaths()
                        opt.ResetState(oldState, oldRoutes, oldLinks)

                        return &InsufficientResourceError{ResourceType:"PathBandwidth", NodeId:depNode  +":" + nodeId}, currentAssignment
 
                    }
                    glog.Infof("check bw for comps %s to %s  on nodes %s and %s\n", compId, component.ComponentId, nId,nodeId ) 
                    glog.Infof("bottleneck link is %s to %s bw = %d\n", bottleneckLink.Src, bottleneckLink.Dst, bottleneckBw)
                    bottleneckLink.BwInUse += depCompBw
                    path.SetPathBw(bottleneckLink.BwInUse)
                    opt.Links[bottleneckLink.Src][bottleneckLink.Dst] = bottleneckLink
                    
                    //opt.UpdatePaths()
                    glog.Info("Updating reverse link usage")
                    reversePath, _ := opt.Routes[nodeId][nId]
                    _, reverseLink := reversePath.FindBottleneckBw()
                    reverseLink.BwInUse += depCompBw
                    reversePath.SetPathBw(reverseLink.BwInUse)
                    opt.Routes[nodeId][nId] = reversePath
                    //opt.UpdatePaths()
                    opt.Links[reverseLink.Src][reverseLink.Dst] = reverseLink
                    opt.UpdatePaths()  
                }

            }
        }
        opt.Nodes[nodeId] = n
        glog.Info("added assignment")
        opt.LogState()
        return nil, currentAssignment
    }
    return err, currentAssignment
}

func (opt *OptimalScheduler) SchedulerHelper(app Application, currentAssignment AppCompAssignment) bool{
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
    currentAssignment := make(AppCompAssignment, 0)
    currentAssignment[app.AppId] = make(map[string]string, 0)
    oldState, oldRoutes, oldLinks := opt.CopyState()
    possible := opt.SchedulerHelper(app, currentAssignment)
    fmt.Printf("is possible for app %s to be scheduled = %s\n", app.AppId, strconv.FormatBool(possible))
    if !possible {
        opt.ResetState(oldState, oldRoutes, oldLinks)
    } 
    opt.PrintState()
    opt.PrintAssignments()
}

