package meshscheduler

import (
    "fmt"
    "github.com/golang/glog"
    "strconv"
    "sort"
)

type MaxBwScheduler struct {
    BaseScheduler
}


func NewMaxBwScheduler()(*MaxBwScheduler) {
    return &MaxBwScheduler{}
}

func (opt *MaxBwScheduler) InitScheduler(nodes NodeMap, routes RouteMap, links LinkMap) {
   
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

func (opt *MaxBwScheduler) CheckFit(comp Component, nodeId string, nodes NodeMap, links LinkMap) (bool, error) {
    nodeState, exists := nodes[nodeId]
    //glog.Infof("node %s cpu available %d, comp %s needs %d\n", nodeId,  nodeState.CpuCapacity -nodeState.CpuInUse, comp.ComponentId, comp.Cpu)
    //glog.Infof("node %s memory available %d, comd %s needs %d\n", nodeId, nodeState.MemoryCapacity -nodeState.MemoryInUse, comp.ComponentId, comp.Memory)


	if !exists {
		return false, &NotFoundError{Msg: "node " + nodeId + " not found"}
	}

	if nodeState.CpuInUse+comp.Cpu > nodeState.CpuCapacity {
		return false, &InsufficientResourceError{ResourceType: "CPU", NodeId: nodeId}
	} 

    if nodeState.MemoryInUse+comp.Memory > nodeState.MemoryCapacity {
		return false, &InsufficientResourceError{ResourceType: "Memory", NodeId: nodeId}
	}
    totalBw := 0.0
    for dst, _ := range links[nodeId]{
        totalBw += links[nodeId][dst].BwCapacity - links[nodeId][dst].BwInUse
    }
    bwNeeded := 0.0
    for _, bw := range comp.Bandwidth{
        bwNeeded += bw
    }
    if bwNeeded > totalBw{
		return false, &InsufficientResourceError{ResourceType: "Bandwidth", NodeId: nodeId}
	
    }
    return true, nil
}

func (opt *MaxBwScheduler) SchedulerHelper(app Application, currentAssignment AppCompAssignment, nodes NodeMap, routes RouteMap, links LinkMap) (bool, AppCompAssignment, NodeMap, LinkMap, RouteMap){
    appAssignment, _ := currentAssignment[app.AppId]
    if len(appAssignment) == len(app.Components){
        return true, deepCopy(currentAssignment), opt.CopyNodes(nodes), links, routes
    }
    nodeOrder := opt.GetNodeOrder(nodes, links)
    compOrder := opt.GetCompOrder(app.Components)
     for _, compId := range compOrder{
        oldRoutes, oldLinks := opt.CopyRoutes(routes, links)
        oldNodes := opt.CopyNodes(nodes)
        comp, _ := app.Components[compId] 
        _, exists := currentAssignment[app.AppId][compId]
        if exists {
            continue
        }
        for _, nodeId := range nodeOrder {
            glog.Infof("node Id %s comp id %s cur assignments %d\n", nodeId, compId, len(appAssignment))
            _, exists := currentAssignment[app.AppId][compId]
            if exists {
                break
            } 
            fit, err := opt.CheckFit(comp, nodeId, nodes, links)
            if !fit || err != nil{
                 glog.Infof("%s", err)
                continue
            }
            err,nodes, links, routes := opt.MakeAssignment(nodeId, compId, app, nodes, routes, links, currentAssignment)
            if err != nil{
                glog.Infof("Insufficient bw resources for %s on %s\n", nodeId, compId)
                continue
            }
            glog.Infof("node id %s comp id %s works, now node has %d \n", nodeId, compId, nodes[nodeId].CpuInUse)
            currentAssignment[app.AppId][compId] = nodeId
            possible, currentAssignment, nodes,  links, routes := opt.SchedulerHelper(app, currentAssignment, nodes, routes, links)
            if possible && len(currentAssignment[app.AppId])== len(app.Components){
                glog.Infof("found assignment %d\n", len(currentAssignment[app.AppId]))
                return true, currentAssignment, nodes, links, routes
            } else {
                delete(currentAssignment[app.AppId], compId)
                routes, links = opt.CopyRoutes(oldRoutes, oldLinks)
                nodes = opt.CopyNodes(oldNodes)
                    
            }
        }
    }
    return false, currentAssignment, nodes, links, routes
}
func (opt *MaxBwScheduler) MakeAssignment(nodeId string, componentId string, app Application, nodes NodeMap, routes RouteMap, links LinkMap, assignment AppCompAssignment) (error,  NodeMap, LinkMap, RouteMap){
    component, _ := app.Components[componentId]
    tmpnodes := opt.CopyNodes(nodes)
    tmproutes, tmplinks := opt.CopyRoutes(routes, links)
            
    //glog.Infof("Test comp %s on node %s, assignments =%d\n", component.ComponentId, nodeId, len(assignment[app.AppId]))
    n, _ := tmpnodes[nodeId] 
    n.CpuInUse += component.Cpu
    n.MemoryInUse += component.Memory
    tmpnodes[nodeId] = n
    // can this node accommodate all bw contraints of this component to the existing component
    for dependency, bw := range component.Bandwidth {
        depNode, exists := assignment[app.AppId][dependency]
        if exists {
            path, exists := tmproutes[nodeId][depNode]
            if !exists {
                return &InsufficientResourceError{ResourceType:"PathBandwidth", NodeId:depNode  +":" + nodeId}, nodes, links, routes
            }
            bottleneckBw, bottleneckLink := path.FindBottleneckBw()
            glog.Infof("comp %s bw %f node %s-%s available %f",  dependency, bw, nodeId, depNode, bottleneckBw)
            if bottleneckBw  >= bw {
                bottleneckLink.BwInUse += bw
                path.SetPathBw(bottleneckLink.BwInUse)
                tmproutes[nodeId][depNode] = path
                tmplinks[bottleneckLink.Src][bottleneckLink.Dst] = bottleneckLink
                reversePath, _ := tmproutes[depNode][nodeId]
                _, reverseLink:= reversePath.FindBottleneckBw()
                reverseLink.BwInUse += bw
                reversePath.SetPathBw(reverseLink.BwInUse)
                opt.UpdatePaths(links, routes)
                tmproutes[depNode][nodeId] = reversePath
                tmplinks[reverseLink.Src][reverseLink.Dst] = reverseLink
                glog.Infof("updating reverse path %s to %s usage = %f\n", depNode, nodeId, tmproutes[nodeId][depNode].BwInUse)
                } else {

                return &InsufficientResourceError{ResourceType:"PathBandwidth", NodeId:nodeId}, nodes, links, routes
            }
        }

    }
    glog.Infof("Updated %s to all deps", component.ComponentId)

    // can all the already scheduled dependencies to this component be satisfied by this node
    appAssignment := assignment[app.AppId]
    for compId, nId := range appAssignment {
        compDeps, _ := app.Components[compId]
        for dep, _ := range compDeps.Bandwidth{
            depNode, _ := appAssignment[dep]
            //glog.Infof("test dep %s for %sn", dep,  compId)
            //if !exists {
            //    continue
            //}
            if dep == component.ComponentId {
                
                path, exists := tmproutes[nId][nodeId]
                glog.Infof("src = %s dst = %s bw in use =%f available=%f\n", nId, nodeId, path.BwInUse, path.BwCapacity)
                if !exists {
                    return &InsufficientResourceError{ResourceType:"PathBandwidth", NodeId:depNode  +":" + nodeId}, nodes,  links, routes
                }
                depCompBw, _ := app.Components[compId].Bandwidth[dep]
                bottleneckBw, bottleneckLink := path.FindBottleneckBw()
                if bottleneckBw - depCompBw < 0{
                    return &InsufficientResourceError{ResourceType:"PathBandwidth", NodeId:depNode  +":" + nodeId}, nodes, links, routes
 
                }
                bottleneckLink.BwInUse += depCompBw
                path.SetPathBw(bottleneckLink.BwInUse)
                tmplinks[bottleneckLink.Src][bottleneckLink.Dst] = bottleneckLink
                
                //opt.UpdatePaths()
                //glog.Info("Updating reverse link usage")
                reversePath, _ := tmproutes[nodeId][nId]
                _, reverseLink := reversePath.FindBottleneckBw()
                reverseLink.BwInUse += depCompBw
                reversePath.SetPathBw(reverseLink.BwInUse)
                tmproutes[nodeId][nId] = reversePath
                opt.UpdatePaths(tmplinks, tmproutes)
                tmplinks[reverseLink.Src][reverseLink.Dst] = reverseLink
            }

        }
    }
    nodes = tmpnodes
    routes, links = tmproutes, tmplinks
    return nil, nodes, links, routes
}

func (opt *MaxBwScheduler) GetCompOrder(comps map[string]Component)[]string{
    type CompTotalBw struct{
        compId string
        bw      float64
        degree int
    }
    compTotalBw := make([]CompTotalBw, 0)
    for compId, comp := range comps {
        bwSum := 0.0
        for _, bw := range comp.Bandwidth {
           bwSum += bw
        }
    
        compTotalBw = append(compTotalBw, CompTotalBw{compId:compId, bw: bwSum, degree:len(comp.Bandwidth)})
    }
    sort.Slice(compTotalBw, func(i int, j int) bool{
       if compTotalBw[i].degree >= compTotalBw[j].degree{
            return true
       }
        return compTotalBw[i].bw >= compTotalBw[j].bw
    })
    compOrder := make([]string, 0)
    for _, compBw := range compTotalBw{
        compOrder = append(compOrder, compBw.compId)
    }
    return compOrder
}


func (opt *MaxBwScheduler) GetNodeOrder(nodes NodeMap, links LinkMap)[]string{
    type NodeTotalBw struct{
        nodeId string
        bw      float64
        degree int
    }
    nodeTotalBw := make([]NodeTotalBw, 0)
    for node, _ := range nodes {
        bwSum := 0.0
        for _, link := range links[node] {
           bwSum += link.BwCapacity 
        }
        nodeTotalBw = append(nodeTotalBw, NodeTotalBw{nodeId:node, bw: bwSum, degree:len(links[node])})
    }
    sort.Slice(nodeTotalBw, func(i int, j int) bool{
        if nodeTotalBw[i].degree >= nodeTotalBw[j].degree {
            return true
        }
        return false
    })
    nodeOrder := make([]string, 0)
    for _, nodeBw := range nodeTotalBw{
        nodeOrder = append(nodeOrder, nodeBw.nodeId)
    }
    return nodeOrder
}

func (opt *MaxBwScheduler) Schedule(app Application) {
    currentAssignment := make(AppCompAssignment, 0)
    currentAssignment[app.AppId] = make(map[string]string, 0)
    oldState, oldRoutes, oldLinks := opt.CopyState()
    possible, currentAssignment, nodes, links, routes := opt.SchedulerHelper(app, currentAssignment, oldState, oldRoutes, oldLinks)
    if possible {
        opt.Nodes, opt.Links, opt.Routes = nodes, links, routes
        opt.UpdatePaths(opt.Links, opt.Routes)
        opt.Assignments[app.AppId] = currentAssignment[app.AppId]
    }
    fmt.Printf("is possible for app %s to be scheduled = %s\n", app.AppId, strconv.FormatBool(possible))
}


