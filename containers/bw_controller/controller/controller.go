package bw_controller

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Controller struct {
	promClient   *PromClient
	netmonClient *NetmonClient
	kubeClient   *KubeClient
	pods         PodSet
	podDepActual PodDeps
	podDepReq    PodDeps
	nodes        map[string]string // ip -> name map
	linksFree    LinkSet
	pathsFree    PathSet
	pathsUsed    TrafficSet
	metrics      []string
}

func NewController(promClient *PromClient, netmonClient *NetmonClient, kubeClient *KubeClient) *Controller {
	controller := &Controller{promClient: promClient, netmonClient: netmonClient, kubeClient: kubeClient}
	controller.podDepReq = make(PodDeps, 0)
	controller.podDepActual = make(PodDeps, 0)
	controller.pods = make(PodSet, 0)
	controller.nodes = make(map[string]string, 0)
	controller.linksFree = make(LinkSet, 0)
	controller.pathsFree = make(PathSet, 0)
	controller.pathsUsed = make(TrafficSet, 0)
	return controller
}

// Get list of k8s nodes
func (controller *Controller) UpdateNodes() {
	nodeList, err := controller.kubeClient.GetNodes()
	if err != nil {
		logger("could not get node list")
	}
	for _, node := range nodeList.Items {
		nodeName := node.Metadata.Name
		nodeIp := node.Metadata.Annotations["flannel.alpha.coreos.com/public-ip"]
		controller.nodes[nodeIp] = nodeName
		fmt.Printf("node name = %s node Ip =%s\n", nodeName, nodeIp)
	}
}
func (controller *Controller) UpdatePodMetrics() {
	_, podDeps := controller.promClient.GetPodMetrics()
	for src, deps := range podDeps {
		fmt.Printf("src = %s\n", src)
		_, exists := controller.pods[src]
		if !exists {
			continue
		}
		fmt.Printf("controller knows pod %s\n", src)
		podReqs, reqExists := controller.podDepReq[src]
		if !reqExists {
			continue
		}
		podActuals, actualExists := controller.podDepActual[src]
		if !actualExists {
			podActuals = make(map[string]PodDependency, 0)
		}
		fmt.Println("Process deps for pod %s\n", src)
		for dst, podDep := range deps {
			_, exists := podReqs[dst]
			if !exists {
				continue
			}
			podActual, exists := podActuals[dst]
			if !exists {
				podActual = podDep
			}
			fmt.Printf("Got actual %s -> %s bw = %f\n", src, dst, podDep.bandwidth)
			podActual.bandwidth = podDep.bandwidth
			podActuals[dst] = podActual
		}
		controller.podDepActual[src] = podActuals
	}
}

// Update network bw available between each pair of nodes
func (controller *Controller) UpdateNetMetrics() {
	links, paths, traffics := controller.netmonClient.GetStats()
	for src, dstLinks := range links {
		for dst, link := range dstLinks {

			srcNode, exists := controller.nodes[src]
			dstNode, dexists := controller.nodes[dst]
			if !exists || !dexists {
				fmt.Println("either source or destination dopn't exist!")
			}
			fmt.Printf("src = %s dst = %s cap = %f\n", srcNode, dstNode, link.bandwidth)
			_, lExists := controller.linksFree[srcNode]
			if !lExists {
				controller.linksFree[srcNode] = make(map[string]Link, 0)

			}
			srcLinks, _ := controller.linksFree[srcNode]
			srcLinks[dstNode] = link
			controller.linksFree[srcNode] = srcLinks
		}
	}
	for src, dstPaths := range paths {
		for dst, path := range dstPaths {

			srcNode, exists := controller.nodes[src]
			dstNode, dexists := controller.nodes[dst]
			if !exists || !dexists {
				fmt.Println("either source or destination dopn't exist!")
			}
			fmt.Printf("src = %s dst = %s cap = %f\n", srcNode, dstNode, path.bandwidth)
			_, pExists := controller.pathsFree[srcNode]
			if !pExists {
				controller.pathsFree[srcNode] = make(map[string]Path, 0)

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
				fmt.Println("either source or destination dopn't exist!")
			}
			fmt.Printf("src = %s dst = %s cap = %f\n", srcNode, dstNode, traffic.bytes)
			srcTraf, tExists := controller.pathsUsed[srcNode]
			if !tExists {
				controller.pathsUsed[srcNode] = make(map[string]Traffic, 0)
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
	fmt.Printf("Got pods from %d namespaces\n", len(podLists))

	for _, podList := range podLists {
		for _, kubePod := range podList.Items {
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

				if strings.Contains(k, "bw") || strings.Contains(k, "latency") {
					vals := strings.Split(k, ".")
					if len(vals) < 3 {
						logger(fmt.Sprintf("ERROR: Incorrect annotation format for pod dependency %s", k))
					}

					dependeeName := vals[1]
					qtyName := vals[0]
					qty, err := strconv.ParseFloat(v, 64)
					if err != nil {
						logger("error parsing float value " + v)
					}
					_, isPodPresent := podSet[dependeeName]
					if !isPodPresent {
						logger(fmt.Sprintf("ERROR: Dependee pod %s not found", dependeeName))
					} else {
						dep := PodDependency{source: podName, destination: dependeeName, bandwidth: 0, latency: 0}
						podDep, exists := podDeps[podName][dependeeName]
						if !exists {
							podDep = dep
						}
						if qtyName == "bw" {
							podDep.bandwidth = qty
						} else {
							podDep.latency = qty
						}
						podDeps[podName][dependeeName] = podDep
						fmt.Printf("Got dependency %s -> %s qty %s = %f\n", podName, dependeeName, qtyName, qty)
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
	fmt.Printf("Added %d pod deps\n", podDepCt)

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
	shortfall := make(map[string]float64, 0)
	totalShortfall := 0.0
	if exists {
		for dst, dep := range podDeps {
			dstPod, _ := controller.pods[dst]
			actual, _ := controller.podDepActual[pod.podName][dst]
			dstNode := dstPod.deployedNode
			if dstNode != pod.deployedNode && actual.bandwidth+bwAvailable[pod.deployedNode][dstNode] < dep.bandwidth {
				diff := dep.bandwidth - actual.bandwidth + bwAvailable[pod.deployedNode][dstNode]
				logger(fmt.Sprintf("Node %s src pod %s dst pod %s dst Node %s diff %f\n", pod.deployedNode, pod.podName, dst, dstNode, diff))
				shortfall[dstNode] = diff

				totalShortfall += diff
			}
		}
		if len(shortfall) > 0 {
			return false, shortfall, totalShortfall
		}
	}
	return true, nil, 0
}

func (controller *Controller) findPodsToReschedule(bwNeeded map[string]map[string]float64,
	bwAvailable map[string]map[string]float64,
	node string) (bool, Pod) {
	pList := make(PairList, 0)

	podList := controller.getPodsOnNode(node)
	for _, pod := range podList {
		satisfied, _, totalShortfall := controller.CheckPodReqSatisfiedOne(bwNeeded, bwAvailable, pod)
		logger(fmt.Sprintf("pod = %s shortfall=%f node=%s\n", pod.podName, totalShortfall, pod.deployedNode))
		if !satisfied {
			pList = append(pList, Pair{Key: pod.podName, Value: totalShortfall})
		}
	}
	if len(pList) == 0 {
		return false, Pod{}
	}
	sort.Sort(pList)
	podToReschedule, _ := controller.pods[pList[0].Key]
	return true, podToReschedule
}

// evalauate if the current deployment satisfies the pod requirement.
// Make a list of pods which need to be rescheduled
// For all the pods that the controller knows of, find the bw used by the pod and the bw available to the pod
// Check if the node has sufficient bw to all dependees of the pod
func (controller *Controller) EvaluateDeployment() {
	fmt.Printf("REQ\n")
	bwNeeded := make(map[string]map[string]float64, 0)
	bwAvailable := make(map[string]map[string]float64, 0)
	for src, srcPaths := range controller.pathsFree {
		for dst, _ := range srcPaths {
			logger(fmt.Sprintf("src = %s dst = %s\n", src, dst))
		}
	}
	// initialize bw needed at each node
	for src, podDeps := range controller.podDepReq {
		for dst, podReq := range podDeps {
			podActual, exists := controller.podDepActual[src][dst]
			if exists {
				logger(fmt.Sprintf("Pod dependency src = %s dst = %s bw req = %f actual = %f\n", src, dst, podReq.bandwidth, podActual.bandwidth))
				srcPod, _ := controller.pods[src]
				dstPod, _ := controller.pods[dst]
				logger(fmt.Sprintf("node for %s = %s and %s = %s \n", src, srcPod.deployedNode, dst, dstPod.deployedNode))
				if srcPod.deployedNode == dstPod.deployedNode {
					logger(fmt.Sprintf("pod %s and %s on same node, skipping", src, dst))
					continue
				}
				_, exists := bwNeeded[srcPod.deployedNode]
				if !exists {
					bwNeeded[srcPod.deployedNode] = make(map[string]float64, 0)

				}
				_, exists = bwNeeded[srcPod.deployedNode][dstPod.deployedNode]
				if !exists {
					bwNeeded[srcPod.deployedNode][dstPod.deployedNode] = 0
				}
				bwNeeded[srcPod.deployedNode][dstPod.deployedNode] += podReq.bandwidth

				srcNodeBws, exists := controller.pathsFree[srcPod.deployedNode]
				if exists {
					_, availExists := bwAvailable[srcPod.deployedNode]
					if !availExists {
						bwAvailable[srcPod.deployedNode] = make(map[string]float64, 0)
						bwAvailable[srcPod.deployedNode][dstPod.deployedNode] = 0
					}
					_, dExists := controller.pathsFree[dstPod.deployedNode]
					if dExists {
						bwAvailable[srcPod.deployedNode][dstPod.deployedNode] = srcNodeBws[dstPod.deployedNode].bandwidth
					}
				}
			}
		}
	}

	// compute bw free by subtracting bw used by pods deployed on each node
	for src, podDeps := range controller.podDepReq {
		for dst, _ := range podDeps {
			podActual, exists := controller.podDepActual[src][dst]
			if exists {
				srcPod, _ := controller.pods[src]
				dstPod, _ := controller.pods[dst]
				if srcPod.deployedNode == dstPod.deployedNode {
					continue
				}
				srcBws, exists := bwAvailable[srcPod.deployedNode]
				if exists {
					bw, dExists := srcBws[dstPod.deployedNode]
					if dExists {
						bw -= podActual.bandwidth
						srcBws[dstPod.deployedNode] = bw
					}
				}
				bwAvailable[srcPod.deployedNode] = srcBws
			}
		}
	}
	nodes := controller.getNodes()
	for _, node := range nodes {
		needToReschedule, pod := controller.findPodsToReschedule(bwNeeded, bwAvailable, node)
		if needToReschedule {
			fmt.Printf("Pod %s ns %s needs to be rescheduled\n", pod.podName, pod.namespace)
			controller.kubeClient.DeletePod(pod.podId, pod.namespace)
		}
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
			controller.UpdateNetMetrics()
			controller.EvaluateDeployment()
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
