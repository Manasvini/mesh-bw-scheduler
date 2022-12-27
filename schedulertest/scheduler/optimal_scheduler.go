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

