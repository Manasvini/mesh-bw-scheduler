package meshscheduler

import (
    "fmt"
    "github.com/golang/glog"
    "strconv"
    "math/rand"
    //"time"
    "math"
    "reflect"
    "sort"
)

type SimulatedAnnealingScheduler struct {
    BaseScheduler
}


func NewSimulatedAnnealingScheduler()(*SimulatedAnnealingScheduler) {
    return &SimulatedAnnealingScheduler{}
}

func (opt *SimulatedAnnealingScheduler) InitScheduler(nodes NodeMap, routes RouteMap, links LinkMap) {
   
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

func (opt *SimulatedAnnealingScheduler) CheckFit(comp Component, nodeId string, nodes NodeMap, links LinkMap) (bool, error) {
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

func (opt *SimulatedAnnealingScheduler) makeInitialAssignment(app Application, nodes NodeMap, links LinkMap) AppCompAssignment{
    //s := rand.NewSource(time.Now().Unix())
    //r := rand.New(s) // initialize local pseudorandom generator 

    assignment := make(AppCompAssignment, 0)
    assignment[app.AppId] = make(map[string]string, 0)
    nodeList := opt.GetNodeOrder(nodes, links)
    compOrder := opt.GetCompOrder(app.Components)
    for i, comp := range compOrder{
        assignment[app.AppId][comp.compId] =  nodeList[i % len(nodeList)].nodeId
        glog.Infof("assign comp %s to node %s \n", comp.compId, nodeList[i% len(nodeList)].nodeId)
        //nodeList[idx].bw -= compOrder[i].bw
        //if nodeList[idx].bw <= 0{
        //    idx += 1
        //}
        //if idx == len(nodeList){
        //    break
        //}
    }
    return assignment
}

func (opt *SimulatedAnnealingScheduler) computeCost(app Application, assignment AppCompAssignment, nodes NodeMap, links LinkMap, routes RouteMap) ([]string, []string, NodeMap, LinkMap, RouteMap){
    violatedComps := make([]string, 0)
    scheduledComps := make([]string, 0)
    for compId, nodeId := range assignment[app.AppId]{
        _, err1 := opt.CheckFit(app.Components[compId], nodeId, nodes, links)
        err2, newnodes, newlinks, newroutes := opt.MakeAssignment(nodeId, compId, app, nodes, routes, links, assignment)
        if err1 != nil || err2 != nil{
            violatedComps = append(violatedComps, compId)
        } else{
            scheduledComps = append(scheduledComps, compId)
        }
        nodes = newnodes
        links, routes = newlinks, newroutes
    }
    glog.Infof("scheduled=%d violated=%d\n", len(scheduledComps), len(violatedComps))
    return scheduledComps, violatedComps, nodes, links, routes
}

func (opt *SimulatedAnnealingScheduler) findNewState(app Application,  compId string, nodes NodeMap, links LinkMap) (string, error){
    totalBwNeeded := 0.0
    comp, _ := app.Components[compId]
    for _, bw := range app.Components[compId].Bandwidth{
        totalBwNeeded += bw
    }
    assignNode := ""
    nodeList := make([]string, 0)
    for nodeId, _ := range nodes{
        nodeList = append(nodeList, nodeId)
    }
    rand.Shuffle(len(nodeList), func(i, j int) {
        nodeList[i], nodeList[j] = nodeList[j], nodeList[i]
    })
    for _, nodeId := range nodeList{
        node := nodes[nodeId]
        totalAvailable := 0.0
        for _, link := range links[nodeId]{
            bwAvailable := link.BwCapacity - link.BwInUse
            totalAvailable += bwAvailable
        }
        //_, exists :=  badNodes[nodeId]
        if totalAvailable >= totalBwNeeded &&  comp.Cpu <= node.CpuCapacity - node.CpuInUse &&  comp.Memory <= node.MemoryCapacity - node.MemoryInUse{
            assignNode = nodeId

            return assignNode, nil
        }
    }
    return "", &InsufficientResourceError{ResourceType:"Bandwidth", NodeId:"No node"}

}
func(opt *SimulatedAnnealingScheduler) findNeighbor(assignment AppCompAssignment, app Application, nodes NodeMap, links LinkMap) AppCompAssignment{
    newAssignment := deepCopy(assignment)
    curAssignment, _ := newAssignment[app.AppId]
    keys := reflect.ValueOf(curAssignment).MapKeys()
    for i := 0; i < len(keys)-1; i++{
        assignKey0,_ := curAssignment[keys[i].Interface().(string)]
        //newNode, _ := opt.findNewState(app, keys[i].Interface().(string), nodes, links)
        if len(keys) > 1 {
            //assignKey1 := newNode
            assignKey1, _ := curAssignment[keys[1].Interface().(string)]
            curAssignment[keys[i].Interface().(string)] = assignKey1
            curAssignment[keys[i+1].Interface().(string)] = assignKey0
            newAssignment[app.AppId] = curAssignment
    
            glog.Infof("comp %s old node=%s new node=%s\n", keys[0].Interface().(string), assignKey0, assignKey1)
         
            return newAssignment
        }
    }
    return newAssignment
}
type CompTotalBw struct{
    compId string
    bw      float64
    degree int
}
func (opt *SimulatedAnnealingScheduler) GetCompOrder(comps map[string]Component)[]CompTotalBw{
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
    return compTotalBw
}

type NodeTotalBw struct{
        nodeId string
        bw      float64
        degree int
}
func (opt *SimulatedAnnealingScheduler) GetNodeOrder(nodes NodeMap, links LinkMap)[]NodeTotalBw{
    nodeTotalBw := make([]NodeTotalBw, 0)
    for node, _ := range nodes {
        bwSum := 0.0
        for dst, link := range links[node] {
            if dst == node{
                continue
            }
            bwSum += link.BwCapacity 
        }
        glog.Infof("node id = %s bw = %f\n", node, bwSum)
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

func (opt *SimulatedAnnealingScheduler) SchedulerHelper(app Application, nodes NodeMap, routes RouteMap, links LinkMap, maxSteps int) (bool, AppCompAssignment, NodeMap, LinkMap, RouteMap){
    temperatureBegin := 5.0e+3
    temperature := temperatureBegin
    temperatureEnd := .1
    coolingFactor := .99
    min := 0.0
    max := 1.0
    minCost := len(app.Components)
    assignment := opt.makeInitialAssignment(app, nodes, links)
    for {
        tmpNodes := opt.CopyNodes(nodes)
        tmpRoutes, tmpLinks := opt.CopyRoutes(routes, links)
        _, violations, oldNodes, oldLinks, oldRoutes := opt.computeCost(app, deepCopy(assignment), tmpNodes, tmpLinks, tmpRoutes)
        cost := len(violations)
        glog.Infof("cost = %d temp = %f\n", cost, temperature)
        if cost == 0{
            return true, assignment, tmpNodes, tmpLinks, tmpRoutes
        }
        //if cost == len(app.Components) {
        //    assignment = opt.makeInitialAssignment(app, nodes)
        ////     
        //} 
        if temperature < temperatureEnd{
            break
        }
        newAssignment := opt.findNeighbor(assignment, app, nodes, links) 
        _, newViolations, newNodes, newLinks, newRoutes := opt.computeCost(app, deepCopy(newAssignment), tmpNodes, tmpLinks, tmpRoutes)
        newCost := len(newViolations)
        diff := 1.0/ float64(newCost - cost)
        prob := min + rand.Float64() * (max - min)
        nodes = oldNodes
        links = oldLinks
        routes = oldRoutes
        if diff >= 0 ||  math.Exp( -diff / temperature ) > prob{
            assignment = newAssignment
            nodes = newNodes
            links = newLinks
            routes = newRoutes
        }
        if cost < minCost {
            nodes = tmpNodes
            links = tmpLinks
            routes = tmpRoutes
            minCost = cost
            assignment = newAssignment
        }
        temperature *= coolingFactor
    }
    fmt.Printf("min cost is %d\n", minCost)
    return false,nil, nodes, links, routes
}
func (opt *SimulatedAnnealingScheduler) MakeAssignment(nodeId string, componentId string, app Application, nodes NodeMap, routes RouteMap, links LinkMap, assignment AppCompAssignment) (error,  NodeMap, LinkMap, RouteMap){
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


func (opt *SimulatedAnnealingScheduler) Schedule(app Application) {
    currentAssignment := make(AppCompAssignment, 0)
    currentAssignment[app.AppId] = make(map[string]string, 0)
    oldState, oldRoutes, oldLinks := opt.CopyState()
    possible, currentAssignment, nodes, links, routes := opt.SchedulerHelper(app,  oldState, oldRoutes, oldLinks, 10000)
    if possible {
        opt.Nodes, opt.Links, opt.Routes = nodes, links, routes
        opt.UpdatePaths(opt.Links, opt.Routes)
        opt.Assignments[app.AppId] = currentAssignment[app.AppId]
    }
    fmt.Printf("is possible for app %s to be scheduled = %s\n", app.AppId, strconv.FormatBool(possible))
}


