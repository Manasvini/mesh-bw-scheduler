package meshscheduler

import (
    "fmt"
    "github.com/golang/glog"
    "strconv"
    "math/rand"
    "time"
)

type TabuSearchScheduler struct {
    BaseScheduler
}


func NewTabuSearchScheduler()(*TabuSearchScheduler) {
    return &TabuSearchScheduler{}
}

func (opt *TabuSearchScheduler) InitScheduler(nodes NodeMap, routes RouteMap, links LinkMap) {
   
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

func (opt *TabuSearchScheduler) CheckFit(comp Component, nodeId string, nodes NodeMap, links LinkMap) (bool, error) {
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

func (opt *TabuSearchScheduler) makeInitialAssignment(app Application, nodes NodeMap, links LinkMap) AppCompAssignment{
    //s := rand.NewSource(time.Now().Unix())
    //r := rand.New(s) // initialize local pseudorandom generator 

    assignment := make(AppCompAssignment, 0)
    assignment[app.AppId] = make(map[string]string, 0)
    nodeList := opt.GetNodeOrder(nodes, links)
    compOrder := opt.GetCompOrder(app.Components)
    idx := 0
    for i, comp := range compOrder{
        assignment[app.AppId][comp.compId] =  nodeList[idx].nodeId
        glog.Infof("assign comp %s to node %s idx=%d\n", comp.compId, nodeList[idx].nodeId, idx)
        nodeList[idx].bw -= compOrder[i].bw
        if nodeList[idx].bw <= 0{
            idx += 1
        }
        if idx == len(nodeList){
            break
        }
    }
    return assignment
}

func (opt *TabuSearchScheduler) computeCostUtility(app Application, assignment AppCompAssignment, nodes NodeMap, links LinkMap, routes RouteMap) (float64, NodeMap, LinkMap, RouteMap){
    overconsumptionCpu := 0.0 
    overconsumptionMem := 0.0
    overconsumptionBw := 0.0
    for compid, nodeid := range assignment[app.AppId]{
        _, err1 := opt.CheckFit(app.Components[compid], nodeid, nodes, links)
        if err1 != nil{
            overconsumptionCpu += 100 *float64(nodes[nodeid].CpuInUse + app.Components[compid].Cpu - nodes[nodeid].CpuCapacity)/float64(nodes[nodeid].CpuCapacity)
            overconsumptionMem += 100*float64(nodes[nodeid].MemoryInUse + app.Components[compid].Memory - nodes[nodeid].MemoryCapacity)/float64(nodes[nodeid].MemoryCapacity)
        glog.Infof("overcons cpu = %f mem=%f\n", overconsumptionCpu, overconsumptionMem)
        }
        if overconsumptionCpu < 0{
            overconsumptionCpu = 0.0
        }
        if overconsumptionMem < 0 {
            overconsumptionMem = 0.0
        }
        err2, newnodes, newlinks, newroutes := opt.MakeAssignment(nodeid, compid, app, nodes, routes, links, assignment)
        if err2 != nil{
            bwOversum := 0.0
            glog.Info(err2)
            for dep, bw := range app.Components[compid].Bandwidth{
                depNode, _ := assignment[app.AppId][dep]
                route, exists := newroutes[nodeid][depNode]
                if !exists {
                    bwOversum += 100.0
                } else{
                    bwOversum += 100.0 *(route.BwInUse + bw - route.BwCapacity) / float64(route.BwCapacity)
                }
                glog.Infof("overcons bw = %f comp = %s dep=%s bw needed=%f avail=%f\n", bwOversum, compid, dep, bw, route.BwCapacity)
            }
            if len(app.Components[compid].Bandwidth) > 0{
                bwOversum /= float64(len(app.Components[compid].Bandwidth))
            }
            overconsumptionBw += bwOversum
        }
        if overconsumptionBw < 0{
            overconsumptionBw = 0.0
        }
        if err1 == nil && err2 == nil{
        
            nodes, links, routes = newnodes, newlinks, newroutes
            glog.Infof("assigned comp %s to node %s\n", compid, nodeid)
            glog.Infof("using node %s cpu=%d mem=%d\n", nodeid, nodes[nodeid].CpuInUse, nodes[nodeid].MemoryInUse)
        }

    }
    glog.Infof("cpu = %f mem=%f bw=%f\n", overconsumptionCpu, overconsumptionMem, overconsumptionBw)
    return (overconsumptionBw + overconsumptionMem + overconsumptionCpu)/(3.0 ), nodes, links, routes
}


func (opt *TabuSearchScheduler) isTabuState(assignment AppCompAssignment, assignments []AppCompAssignment, appId string) bool{
    for idx, assign := range assignments {
        curAssign := assign[appId]
        foundState := true
        for node, comp := range curAssign {
            newNode,  exists := assignment[appId][comp]
            if !exists || newNode != node {
               foundState =false
                break
            }
        }
        if foundState == true {
            glog.Infof("state %d is taboo", idx)
            return true
        }
    }
    return false
}

func(opt *TabuSearchScheduler) findNeighbors(assignment AppCompAssignment, app Application, nodes NodeMap, links LinkMap) []AppCompAssignment{
    tmpAssignment := deepCopy(assignment)
    curAssignment, _ := tmpAssignment[app.AppId]
    nodeSet := make(map[string]bool, 0)
    // current node set
    for _, node := range curAssignment{
        nodeSet[node] = true
    }
    glog.Infof("finding new state")
    // find new node to move some components to such that bw between current node with dependencies and new node is adequate
    nodeOrder := opt.GetNodeOrder(nodes, links)
    newAssignments := make([]AppCompAssignment, 0)
    madeAssignment := false
    for comp, compnode := range curAssignment {
        newAssignment := deepCopy(tmpAssignment)
        rand.Seed(time.Now().UnixNano())
        rand.Shuffle(len(nodeOrder), func(i, j int) {
            nodeOrder[i], nodeOrder[j] = nodeOrder[j], nodeOrder[i]
        })
        for _, node := range nodeOrder{
            _, exists := nodeSet[node.nodeId]
            glog.Infof("test node %s for comp %s\n", node.nodeId, comp)
            if !exists || node.nodeId != compnode{
                compTotalBw := 0.0
                for _, bw := range app.Components[comp].Bandwidth{
                    compTotalBw += bw
                }
                if compTotalBw < node.bw {
                    glog.Infof("new state: comp total bw = %f node bw = %f node = %s comp=%s\n", compTotalBw, node.bw, node.nodeId, comp)
                    newAssignment[app.AppId][comp]=node.nodeId
                
                    madeAssignment = true
                    break
                } 
            }
        }
        if madeAssignment {
            newAssignments = append(newAssignments, newAssignment)
            madeAssignment = false 
            opt.LogAssignmentsHelper(newAssignment)
        }
    }
    glog.Infof("have %d new states\n", len(newAssignments))
    return newAssignments
}

func (opt *TabuSearchScheduler) SchedulerHelper(app Application, nodes NodeMap, routes RouteMap, links LinkMap, maxSteps int) (bool, AppCompAssignment, NodeMap, LinkMap, RouteMap){
    curAssignment := opt.makeInitialAssignment(app, nodes, links)
    overallBestAssignment := deepCopy(curAssignment)
    overallBestCost, nodes, links, routes := opt.computeCostUtility(app, overallBestAssignment, nodes, links, routes)
    tabuList := make([]AppCompAssignment, 0)
    tabuList = append(tabuList, overallBestAssignment)
    numSteps := 0
    oldNodes := opt.CopyNodes(nodes)
    oldRoutes, oldLinks := opt.CopyRoutes(routes, links)
    bestAssignment := overallBestAssignment
    bestCost, bestnodes, bestlinks, bestroutes := opt.computeCostUtility(app, overallBestAssignment, nodes, links, routes) 
       for {
        if numSteps == maxSteps {
            break
        }
        neighbors := opt.findNeighbors(bestAssignment, app, nodes, links)
        //bestCost, bestnodes, bestlinks, bestroutes = opt.computeCostUtility(app, overallBestAssignment, nodes, links, routes) 
        if bestCost == 0.0{
            return true, overallBestAssignment, bestnodes, bestlinks, bestroutes
        }
        for _, curAssign := range neighbors{
            curCost, curnodes, curlinks, curroutes := opt.computeCostUtility(app, curAssign, nodes, links,routes)
            if curCost < bestCost{
                bestCost = curCost
                bestAssignment = curAssign
                bestnodes, bestlinks, bestroutes = curnodes, curlinks, curroutes
            }
        }
        if bestCost < overallBestCost && !opt.isTabuState(bestAssignment, tabuList, app.AppId){
            overallBestCost = bestCost
            nodes, links, routes = bestnodes, bestlinks, bestroutes
            overallBestAssignment = bestAssignment
        }
        tabuList = append(tabuList, bestAssignment)
        numSteps += 1
    
        if len(tabuList) == 50{
            tabuList = tabuList[1:len(tabuList)]
        }
        glog.Infof("step = %d cost = %f overall best =%f tabu list size=%d\n", numSteps, bestCost, overallBestCost, len(tabuList))
    }
    glog.Infof("best cost = %f\n", overallBestCost)
    fmt.Printf("cost=%f\n", overallBestCost)
    if overallBestCost == 0.0{
            return true, overallBestAssignment, bestnodes, bestlinks, bestroutes
    }
   return false, nil, oldNodes, oldLinks, oldRoutes
}

func (opt *TabuSearchScheduler) MakeAssignment(nodeId string, componentId string, app Application, nodes NodeMap, routes RouteMap, links LinkMap, assignment AppCompAssignment) (error,  NodeMap, LinkMap, RouteMap){
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

                return &InsufficientResourceError{ResourceType:"PathBandwidth", NodeId:depNode +":"+ nodeId}, nodes, links, routes
            }
        }

    }
    glog.Infof("Updated %s to all deps", component.ComponentId)

    // can all the already scheduled dependencies to this component be satisfied by this node
    appAssignment := assignment[app.AppId]
    for compId, nId := range appAssignment {
        compDeps, _ := app.Components[compId]
        for dep, bw := range compDeps.Bandwidth{
            depNode, _ := appAssignment[dep]
            //glog.Infof("test dep %s for %sn", dep,  compId)
            //if !exists {
            //    continue
            //}
            if dep == component.ComponentId {
                
                path, exists := tmproutes[nId][nodeId]
                glog.Infof("src = %s dst = %s bw in use =%f available=%f needs=%f\n", nId, nodeId, path.BwInUse, path.BwCapacity, bw)
                if !exists {
                    return &InsufficientResourceError{ResourceType:"PathBandwidth", NodeId:depNode  +":" + nId}, nodes,  links, routes
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


func (opt *TabuSearchScheduler) Schedule(app Application) {
    currentAssignment := make(AppCompAssignment, 0)
    currentAssignment[app.AppId] = make(map[string]string, 0)
    oldState, oldRoutes, oldLinks := opt.CopyState()
    possible, currentAssignment, nodes, links, routes := opt.SchedulerHelper(app,  oldState, oldRoutes, oldLinks, 100)
    if possible {
        opt.Nodes, opt.Links, opt.Routes = nodes, links, routes
        opt.UpdatePaths(opt.Links, opt.Routes)
        opt.Assignments[app.AppId] = currentAssignment[app.AppId]
    }
    fmt.Printf("is possible for app %s to be scheduled = %s\n", app.AppId, strconv.FormatBool(possible))
}


