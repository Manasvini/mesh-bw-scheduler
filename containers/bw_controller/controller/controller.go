package bw_controller

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"math"
	//"io/ioutil"
	//"io"
	"os"
	netmon_client "github.gatech.edu/cs-epl/mesh-bw-scheduler/netmon_client"
)

type Controller struct {
	promClient   *PromClient
	netmonClient *netmon_client.NetmonClient
	kubeClient   *KubeClient
	pods         PodSet
	podDepActual PodDeps
	podDepReq    PodDeps
	nodes        map[string]string // ip -> name map
	linksFree    netmon_client.LinkSet
	pathsFree    netmon_client.PathSet
	pathsUsed    netmon_client.TrafficSet
	metrics      []string
	valuationInterval int64
	namespaceValuationTime map[string]int64
	namespaceAvgUtilization map[string]float64
	utilChangeThreshold	float64
	bwFile	*os.File
	migrationFile	*os.File
	pendingBwUpdate	bool
	headroomReq	map[string]map[string]float32
	headroomThreshold float32
	ipMap		map[string]string
}

func NewController(promClient *PromClient, 
		   netmonClient *netmon_client.NetmonClient, 
		   kubeClient *KubeClient, 
		   valuationInterval int64, 
		   utilChangeThreshold float64, 
		   bwFile string, 
		   migrationFile string, 
		   headroomThreshold float32,
	   	   ipMap map[string]string) *Controller {
	controller := &Controller{promClient: promClient, netmonClient: netmonClient, kubeClient: kubeClient, pendingBwUpdate: false}
	controller.podDepReq = make(PodDeps, 0)
	controller.podDepActual = make(PodDeps, 0)
	controller.pods = make(PodSet, 0)
	controller.nodes = make(map[string]string, 0)
	controller.linksFree = make(netmon_client.LinkSet, 0)
	controller.pathsFree = make(netmon_client.PathSet, 0)
	controller.pathsUsed = make(netmon_client.TrafficSet, 0)
	controller.valuationInterval = valuationInterval
	controller.utilChangeThreshold = utilChangeThreshold
	controller.namespaceValuationTime = make(map[string]int64, 0)
	controller.namespaceAvgUtilization = make(map[string]float64, 0)
	controller.headroomReq = make(map[string]map[string]float32, 0)
	controller.headroomThreshold = headroomThreshold
	controller.bwFile, _ = os.OpenFile(bwFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	controller.migrationFile, _ = os.OpenFile(migrationFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	controller.bwFile.WriteString(fmt.Sprintf("time,src,dst,bw\n"))
	controller.migrationFile.WriteString("time,pod\n")
	controller.ipMap = ipMap
	// intialize state for cluster
	controller.UpdateNodes()
	controller.UpdatePods()
	controller.UpdatePodMetrics()
	controller.UpdateNetMetrics(true)
	
	return controller
}

func (controller *Controller) Shutdown() {
	controller.bwFile.Close()
	controller.migrationFile.Close()
}
// Get list of k8s nodes
func (controller *Controller) UpdateNodes() {
	nodeList, err := controller.kubeClient.GetNodes()
	if err != nil {
		logger("could not get node list")
	}
	for _, node := range nodeList.Items {
		nodeName := node.Metadata.Name
		nodeIp, exists := node.Metadata.Annotations["alpha.kubernetes.io/provided-node-ip"]
		if !exists {
			nodeIp = node.Metadata.Annotations["flannel.alpha.coreos.com/public-ip"]
		}
		controller.nodes[nodeIp] = nodeName
		logger(fmt.Sprintf("node name = %s node Ip =%s\n", nodeName, nodeIp))
	}
}
func (controller *Controller) UpdatePodMetrics() {
	_, podDeps := controller.promClient.GetPodMetrics()
	for src, deps := range podDeps {
		//logger(fmt.Sprintf("src = %s\n", src))
		_, exists := controller.pods[src]
		if !exists {
			continue
		}
		//logger(fmt.Sprintf("controller knows pod %s\n", src))
		podReqs, reqExists := controller.podDepReq[src]
		if !reqExists {
			continue
		}
		podActuals, actualExists := controller.podDepActual[src]
		if !actualExists {
			podActuals = make(map[string]PodDependency, 0)
		}
		//logger(fmt.Sprintf("Process deps for pod %s has %d deps\n", src, len(deps)))
		for dst, podDep := range deps {
			//logger("dst = " + dst)
			_, exists := podReqs[dst]
			if !exists {
				//logger("dst " + dst + "does not exist")
				continue
				//controller.podDepReq[src][dst] = podDep
			}
			podActual, exists := podActuals[dst]
			podActual = podDep
			//logger(fmt.Sprintf("Got actual %s -> %s bw = %f\n", src, dst, podDep.Bandwidth))
			podActual.Bandwidth = 8 * podDep.Bandwidth
			podActual.FractionUsed = podActual.Bandwidth / podReqs[dst].Bandwidth
			podActuals[dst] = podActual
		}
		controller.podDepActual[src] = podActuals
	}
}

// Update network bw available between each pair of nodes
func (controller *Controller) UpdateNetMetrics(isBwUpdate bool) {
	nodemap := controller.ipMap
	var links netmon_client.LinkSet
	var paths netmon_client.PathSet
	var traffics netmon_client.TrafficSet
	if isBwUpdate == true {
		links, paths, traffics = controller.netmonClient.GetStats(nodemap)
	} else {
		links, paths, traffics = controller.netmonClient.GetHeadroomStats(nodemap, controller.headroomReq)
	}

	for src, dstLinks := range links {
		for dst, link := range dstLinks {
			//logger(fmt.Sprintf("src = %s dst = %s", src, dst))
			srcNode, exists := controller.nodes[src]
			dstNode, dexists := controller.nodes[dst]
			if !exists || !dexists {
				logger("either source or destination dopn't exist!")
			}
			logger(fmt.Sprintf("src = %s dst = %s cap = %f\n", srcNode, dstNode, link.Bandwidth))
			_, lExists := controller.linksFree[srcNode]
			if !lExists {
				controller.linksFree[srcNode] = make(map[string]netmon_client.Link, 0)

			}
			srcLinks, _ := controller.linksFree[srcNode]
			srcLinks[dstNode] = link
			controller.linksFree[srcNode] = srcLinks
			if isBwUpdate {
				_, exists = controller.headroomReq[src]
				if !exists {
					controller.headroomReq[src] = make(map[string]float32, 0)
				}
				controller.headroomReq[src][dst] = float32(link.Bandwidth) * controller.headroomThreshold
				logger(fmt.Sprintf("threshold src = %s dst = %s bw = %f", src, dst, controller.headroomReq[src][dst]))
			}
		}
	}
	for src, dstPaths := range paths {
		for dst, path := range dstPaths {
			srcNode, exists := controller.nodes[src]
			dstNode, dexists := controller.nodes[dst]
			if !exists || !dexists {
				continue
			}
			//logger(fmt.Sprintf("src = %s dst = %s cap = %f\n", srcNode, dstNode, path.Bandwidth))
			_, pExists := controller.pathsFree[srcNode]
			if !pExists {
				controller.pathsFree[srcNode] = make(map[string]netmon_client.Path, 0)

			}
			srcPaths, _ := controller.pathsFree[srcNode]
			srcPaths[dstNode] = path
			controller.pathsFree[srcNode] = srcPaths
		}
	}

	for src, dstTrafs := range traffics {
		for dst, traffic := range dstTrafs {

			srcNode, exists := controller.nodes[src]
			dstNode, dexists := controller.nodes[dst]
			if !exists || !dexists {
				continue
			}
			//logger(fmt.Sprintf("src = %s dst = %s used = %f\n", srcNode, dstNode, traffic.Bytes))
			srcTraf, tExists := controller.pathsUsed[srcNode]
			if !tExists {
				controller.pathsUsed[srcNode] = make(map[string]netmon_client.Traffic, 0)
			}
			srcTraf, _ = controller.pathsUsed[srcNode]
			srcTraf[dstNode] = traffic
			controller.pathsUsed[srcNode] = srcTraf
		}
	}

}

func getPodName(podId string) string {
	vals := strings.Split(podId, "-")
	podName := ""
	for i := 0; i < len(vals)-2; i++ {
		podName = podName + vals[i]
		if i < len(vals)-3 {
			podName += "-"
		}
	}
	return podName
}

// Check API server for new pods that were added since we last checked
func (controller *Controller) UpdatePods() {
	podLists := controller.kubeClient.GetPods()
	podSet := make(PodSet, 0)
	podDeps := make(PodDeps, 0)
	logger(fmt.Sprintf("Got pods from %d namespaces\n", len(podLists)))

	for _, podList := range podLists {
		for _, kubePod := range podList.Items {
			//ns := kubePod.Metadata.Namespace
			//_, exists := controller.namespaceValuationTime[ns]
			//if !exists {
			//	controller.namespaceValuationTime[ns] = time.Now().Unix()
			//	controller.namespaceAvgUtilization[ns] = 0.0
			//}
			podName := getPodName(kubePod.Metadata.Name)
			podInfo := Pod{podName: podName, podId: kubePod.Metadata.Name, deployedNode: kubePod.Spec.NodeName, namespace: kubePod.Metadata.Namespace}
			podSet[podName] = podInfo
			//logger(fmt.Sprintf("Got pod %s", kubePod.Metadata.Name))
			podDeps[podName] = make(map[string]PodDependency, 0)
		}
	}
	for _, podList := range podLists {
		for _, kubePod := range podList.Items {
			podName := getPodName(kubePod.Metadata.Name)
			for k, v := range kubePod.Metadata.Annotations {

				if (strings.Contains(k, "bw") || strings.Contains(k, "latency")) && (strings.Contains(k, "dependedby") || strings.Contains(k, "dependson")) {
					vals := strings.Split(k, ".")
					if len(vals) < 3 {
						logger(fmt.Sprintf("ERROR: Incorrect annotation format for pod dependency %s", k))
					}

					dependeeName := vals[1]
					qtyName := vals[2]
					qty, err := strconv.ParseFloat(v, 64)
					if err != nil {
						logger("error parsing float value " + v)
					}
					_, isPodPresent := podSet[dependeeName]
					if !isPodPresent {
						logger(fmt.Sprintf("ERROR: Dependency destination pod %s not found", dependeeName))
					} else {
						dep := PodDependency{Source: podName, Destination: dependeeName, Bandwidth: 0, Latency: 0}
						podDep, exists := podDeps[podName][dependeeName]
						if !exists {
							podDep = dep
						}
						if qtyName == "bw" {
							podDep.Bandwidth = qty
						} else {
							podDep.Latency = qty
						}
						podDeps[podName][dependeeName] = podDep
						//logger(fmt.Sprintf("Got dependency %s -> %s qty %s = %f\n", podName, dependeeName, qtyName, qty))
					}
				} else if strings.Contains(k, "bw") && strings.Contains(k, "all") {
					qty, err := strconv.ParseFloat(v, 64)
					if err != nil {
						logger("error parsing float value " + v)
					}
					logger(fmt.Sprintf("podname %s needs %f overall", podName, qty))
					if strings.Contains(k, "send") {
						podDepSnd := PodDependency{Source:podName, Destination:"all_send", Bandwidth: qty}
						podDeps[podName]["all_send"] = podDepSnd
					} else if strings.Contains(k, "rcv") {
						podDepRcv := PodDependency{Source:podName, Destination:"all_rcv", Bandwidth: qty}
						podDeps[podName]["all_rcv"] = podDepRcv
					}
				}
			}
		}
	}
	for pname, pod := range podSet {
		controller.pods[pname] = pod
		_, cExists := controller.podDepReq[pname]
		if !cExists {
			controller.podDepReq[pname] = make(map[string]PodDependency, 0)
		}
	}
	podDepCt := 0
	for srcPod, dstPods := range podDeps {

		for dstPod, podDep := range dstPods {
			_, exists := controller.podDepReq[srcPod][dstPod]
			if !exists {
				controller.podDepReq[srcPod][dstPod] = podDep
				podDepCt += 1
			}
		}
	}
	logger(fmt.Sprintf("Added %d pod deps\n", podDepCt))

}

func (controller *Controller) getPodsOnNode(nodeName string) []Pod {
	podList := make([]Pod, 0)
	for _, pod := range controller.pods {
		if pod.deployedNode == nodeName {
			podList = append(podList, pod)
		}
	}
	return podList
}

func (controller *Controller) getNodes() []string {
	nodeNames := make([]string, 0)
	for _, node := range controller.nodes {
		nodeNames = append(nodeNames, node)
	}
	return nodeNames
}

func (controller *Controller) CheckPodReqSatisfiedOne(bwNeeded map[string]map[string]float64,
	bwAvailable map[string]map[string]float64,
	pod Pod) (bool, map[string]float64, float64) /* dep pod -> dep bw shortfall, total shortfall*/ {
	podDeps, exists := controller.podDepReq[pod.podName]
	bwUsed := make(map[string]float64, 0)

	totalShortfall := 0.0
	totalUsed := 0.0
	totalNeeded := 0.0
	if exists {
		for dst, dep := range podDeps {
			dstPod, _ := controller.pods[dst]
			actual, _ := controller.podDepActual[pod.podName][dst]
			dstNode := dstPod.deployedNode
			totalUsed += float64(actual.Bandwidth) 
			totalNeeded += float64(dep.Bandwidth)
			logger("src = " + pod.podName + " dst = " + dst)
			if dstNode != pod.deployedNode && (dst != "all_send"  && dst != "all_rcv") {
				//bwAvailable[pod.deployedNode][dstNode] - dep.Bandwidth > 0 && fracUsed >= controller.utilChangeThreshold{
				diff := dep.Bandwidth - actual.Bandwidth + bwAvailable[pod.deployedNode][dstNode]
				logger(fmt.Sprintf("Node %s src pod %s dst pod %s dst Node %s diff %f dep req bw = %f used=%f available = %f\n", pod.deployedNode, pod.podName, dst, dstNode, diff, dep.Bandwidth,actual.Bandwidth,  bwAvailable[pod.deployedNode][dstNode]))
				if _, exists := bwUsed[dstNode]; !exists{
					bwUsed[dstNode] = 0
				} 
				bwUsed[dstNode] += dep.Bandwidth
				
				if diff < 0 {
					totalShortfall += diff
				}
			
				bwAvailable[pod.deployedNode][dstNode] -= dep.Bandwidth
			} else if dst == "all_send"  {
				min_bw := -1.0
				
				for _, bw := range bwAvailable[pod.deployedNode] {
					if min_bw < 0 {
						min_bw = bw
					}else if bw < min_bw {
						min_bw = bw
					}
				}
				diff := dep.Bandwidth - actual.Bandwidth + min_bw
				if diff < 0 {
					totalShortfall += diff
				}
				logger(fmt.Sprintf("node = %s pod = %s diff = %f dep bw = %f used = %f  avail = %f\n", pod.deployedNode, pod.podName, diff, dep.Bandwidth, actual.Bandwidth, min_bw))
			}
		}
		fracUsed := totalUsed / totalNeeded
		logger(fmt.Sprintf("fracUsed = %f", fracUsed))
		if totalShortfall < 0  && fracUsed > controller.utilChangeThreshold{

			return false, bwUsed, totalShortfall
		}
	}
	return true, nil, 0
}

func (controller *Controller) findPodsToReschedule(bwNeeded map[string]map[string]float64,
	bwAvailable map[string]map[string]float64,
	node string) (bool, []Pod) {
	pList := make(PairList, 0)

	podList := controller.getPodsOnNode(node)
	bwThresholdViolated := false
	for _, pod := range podList {
		satisfied, usedBw, totalShortfall := controller.CheckPodReqSatisfiedOne(bwNeeded, bwAvailable, pod)
		logger(fmt.Sprintf("pod = %s shortfall=%f node=%s\n", pod.podName, totalShortfall, pod.deployedNode))
		
		for node, val := range usedBw {
			if pod.deployedNode != node {
				if float32(bwAvailable[pod.deployedNode][node] - val) < controller.headroomReq[pod.deployedNode][node] {
					bwThresholdViolated = true
				}
				bwAvailable[pod.deployedNode][node] -= val

			}
		} 
		if !satisfied || bwThresholdViolated {
			pList = append(pList, Pair{Key: pod.podName, Value: totalShortfall})
			logger("added pod " + pod.podName + " to reschedule")
		}
	}
	podsToReschedule := make([]Pod, 0)
	if len(pList) == 0 {
		return false, podsToReschedule
	}
	sort.Sort(pList)
	logger(fmt.Sprintf("Have %d pods to reschedule initially", len(pList)))
	for idx, p := range pList{
		depExists := false
		for _, otherPod := range pList[0: idx] {
			_, exists := controller.podDepReq[p.Key][otherPod.Key]
			if exists {
				depExists = true
				break
			}
		}
		if !depExists{
			podsToReschedule = append(podsToReschedule, controller.pods[p.Key])
		}
	}
	return true, podsToReschedule
}

func (controller *Controller) ShouldReschedulePods(namespace string) bool {
	// for pods in the same namespace check if the usage is much lesser or greater than a set threshold. On average if most pods are under/overutilizing bandwidth, we reschedule
	avgUtilization := 0.0
	numDeps := 0
	logger("ns is " + namespace)
	
	for podName, podDep := range controller.podDepActual {
		podInfo, _ := controller.pods[podName]
		if podInfo.namespace != namespace {
			continue
		}
		_, exists := controller.namespaceValuationTime[namespace]
		if !exists {
			controller.namespaceValuationTime[namespace] = 0
		}
		ts, _ := controller.namespaceValuationTime[namespace]
		timediff := time.Now().Unix() - ts
		if timediff < controller.valuationInterval {
			logger(fmt.Sprintf("ns %s not due for evaluation, %d seconds have passed", namespace, timediff)) 
			return false
		}
		for otherPod, dep := range podDep {
			logger(fmt.Sprintf("pod %s dep %s usage frac %f", podName, otherPod, dep.FractionUsed))
			avgUtilization += dep.FractionUsed
			numDeps += 1
		}
	}
	avgUtilization /= float64(numDeps)
	logger(fmt.Sprintf("ns = %s avg util = %f", namespace, avgUtilization))
	prevUtilization, _ := controller.namespaceAvgUtilization[namespace]
	controller.namespaceAvgUtilization[namespace] = avgUtilization
	if avgUtilization > 1  || (math.Abs(prevUtilization - avgUtilization) > controller.utilChangeThreshold ) {
	 	logger("relocating ns " + namespace)	
		return true
	}
	return false
}


func (controller *Controller) EvaluateUsage() {
	for ns, _ := range controller.namespaceValuationTime {
		reschedule := controller.ShouldReschedulePods(ns)
		if reschedule {
			for _, podInfo := range controller.pods {
				if podInfo.namespace == ns {
					//controller.kubeClient.DeletePod(podInfo.podId, podInfo.namespace)
					logger("deleting pod " + podInfo.podId)
				}
			}
			controller.namespaceValuationTime[ns] = time.Now().Unix()
		}
	}
}

// evalauate if the current deployment satisfies the pod requirement.
// Make a list of pods which need to be rescheduled
// For all the pods that the controller knows of, find the bw used by the pod and the bw available to the pod
// Check if the node has sufficient bw to all dependees of the pod
func (controller *Controller) EvaluateDeployment() {
	logger("REQ\n")
	bwNeeded := make(map[string]map[string]float64, 0)
	bwAvailable := make(map[string]map[string]float64, 0)
	// calculate available bandwidth
	for src, srcPaths := range controller.pathsFree {
		for dst, bw := range srcPaths {
			controller.bwFile.WriteString(fmt.Sprintf("%d,%s,%s,%f\n", time.Now().Unix(), src, dst, bw.Bandwidth))
			_, availExists := bwAvailable[src]
			if !availExists {
				bwAvailable[src] = make(map[string]float64, 0)
				bwAvailable[src][dst] = 0
			}
			bwAvailable[src][dst] = bw.Bandwidth
			logger(fmt.Sprintf("src = %s dst = %s bw = %f\n", src, dst, bw.Bandwidth))
		}
	}
	// initialize bw needed at each node
	for src, podDeps := range controller.podDepReq {
		podInfo, exists := controller.pods[src]
		if !exists {
			logger("UNKNOWN POD " + src)
			continue
		}
		ns := podInfo.namespace
		nsValTime, exists := controller.namespaceValuationTime[ns]
		if !exists {
			nsValTime = 0
		}
		diff := time.Now().Unix() - nsValTime
		if diff < controller.valuationInterval {
			continue
		}
		for dst, podReq := range podDeps {
			podActual, exists := controller.podDepActual[src][dst]
			//logger(fmt.Sprintf("src = %s dst = %s\n", src, dst))
			// bw needed for each dependency pair
			if exists && (dst != "all_send" && dst != "all_rcv"){
				logger(fmt.Sprintf("Pod dependency src = %s dst = %s bw req = %f actual = %f\n", src, dst, podReq.Bandwidth, podActual.Bandwidth))
				srcPod, _ := controller.pods[src]
				dstPod, _ := controller.pods[dst]
				//logger(fmt.Sprintf("node for %s = %s and %s = %s \n", src, srcPod.deployedNode, dst, dstPod.deployedNode))
				if srcPod.deployedNode == dstPod.deployedNode {
					//logger(fmt.Sprintf("pod %s and %s on same node, skipping", src, dst))
					continue
				}
				_, bwExists := bwNeeded[srcPod.deployedNode]
				if !bwExists {
					bwNeeded[srcPod.deployedNode] = make(map[string]float64, 0)

				}
				_, bwExists = bwNeeded[srcPod.deployedNode][dstPod.deployedNode]
				if !bwExists {
					bwNeeded[srcPod.deployedNode][dstPod.deployedNode] = 0
				}
				bwNeeded[srcPod.deployedNode][dstPod.deployedNode] += podReq.Bandwidth

			} else if exists && (dst == "all_send" || dst == "all_rcv") { // bw needed by a standalone service with clients
				logger("standalone pod " + src)
				srcPod, _ := controller.pods[src]
				_, bwExists :=  bwNeeded[srcPod.deployedNode]
				if !bwExists {
					bwNeeded[srcPod.deployedNode] = make(map[string]float64, 0)
				}
				for node, _ := range controller.pathsFree {
					_, bwExists = bwNeeded[srcPod.deployedNode][node]
					if !bwExists {
						bwNeeded[srcPod.deployedNode][node] = podReq.Bandwidth
					}
					if _, exists := bwNeeded[node]; !exists {
						bwNeeded[node] = make(map[string]float64, 0)
						bwNeeded[node][srcPod.deployedNode] = podReq.Bandwidth
					}
				}

			}

		}
	}
	// check if available bw satifies the requirements
	for src, srcBw := range bwNeeded {
		for dst, bw := range srcBw {
			if _, exists := bwAvailable[src][dst]; exists{
				bwAvailable[src][dst] -= bw
			}
		}
	}
	for src, srcBw := range bwAvailable {
		for dst, bw := range srcBw {
			logger(fmt.Sprintf("src = %s dst = %s remaining bw = %f", src, dst,bw))
		}
	}
	logger(fmt.Sprintf("Got %d nodes", len(bwAvailable)))
	if len(bwAvailable) == 0 {
		logger("No pods to reschedule in any namespace")
		return
	}
	nodes := controller.getNodes()
	numRescheduled := 0
	for _, node := range nodes {
		needToReschedule, pods := controller.findPodsToReschedule(bwNeeded, bwAvailable, node)
		if len(pods) == 0{
			continue
		}
		ts, _ := controller.namespaceValuationTime[pods[0].namespace]
		timediff := time.Now().Unix() - ts
		logger(fmt.Sprintf("time diff = %d", timediff))
		if timediff >= controller.valuationInterval && needToReschedule {
			logger(fmt.Sprintf("%d pods need to be rescheduled from node %s\n", len(pods), node))
			for _, pod := range pods {
				controller.migrationFile.WriteString(fmt.Sprintf("%d,%s\n", time.Now().Unix(), pod.podName))	
				logger("moving pod " + pod.podId)
				controller.kubeClient.DeletePod(pod.podId, pod.namespace)
				controller.namespaceValuationTime[pod.namespace] = time.Now().Unix()
				numRescheduled += 1
			}
		}
	}
	logger(fmt.Sprintf("relocating %d pods", numRescheduled))
	if numRescheduled > 0 {
		controller.UpdateNetMetrics(true) // we want to get link capacities
		controller.pendingBwUpdate = true
	}

}

func (controller *Controller) MonitorState(delay time.Duration) chan bool {
	stop := make(chan bool)
	go func() {
		logger("Monitor started")
		for {

			controller.UpdateNodes()
			controller.UpdatePods()
			controller.UpdatePodMetrics()
			controller.UpdateNetMetrics(controller.pendingBwUpdate)	// by default we only update headroom not total link capacity
			controller.pendingBwUpdate = false
			controller.EvaluateDeployment()
			//controller.EvaluateUsage()
			select {
			case <-time.After(delay):
			case <-stop:
				logger("Monitor stopped")
				return
			}
		}
	}()
	return stop
}
