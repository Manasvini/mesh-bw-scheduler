package bw_controller

import (
	"fmt"
	"strconv"
	"strings"
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
	metrics      []string
}

func NewController(promClient *PromClient, netmonClient *NetmonClient, kubeClient *KubeClient) *Controller {
	controller := &Controller{promClient: promClient, netmonClient: netmonClient, kubeClient: kubeClient}
	controller.podDepReq = make(PodDeps, 0)
	controller.podDepActual = make(PodDeps, 0)
	controller.pods = make(PodSet, 0)
	controller.nodes = make(map[string]string, 0)
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
	controller.promClient.UpdatePodMetrics(&(controller.pods), &(controller.podDepActual))
}

// Update network bw available between each pair of nodes
func (controller *Controller) UpdateNetMetrics() {
	links, _ := controller.netmonClient.GetStats()
	for src, dstLinks := range links {
		for dst, link := range dstLinks {

			srcNode, exists := controller.nodes[src]
			dstNode, dexists := controller.nodes[dst]
			if !exists || !dexists {
				fmt.Println("either source or destination dopn't exist!")
			}
			fmt.Printf("src = %s dst = %s cap = %f\n", srcNode, dstNode, link.bandwidth)
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
			podInfo := Pod{podName: podName, podId: kubePod.Metadata.Name, deployedNode: kubePod.Spec.NodeName}
			podSet[podName] = podInfo
			logger(fmt.Sprintf("Got pod %s", kubePod.Metadata.Name))
			podDeps[podName] = make(map[string]PodDependency, 0)
		}
	}
	for _, podList := range podLists {
		for _, kubePod := range podList.Items {
			podName := getPodName(kubePod.Metadata.Name)
			for k, v := range kubePod.Metadata.Annotations {

				if strings.Contains(k, "dependee") {
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
		_, cExists := controller.pods[pname]
		if !cExists {
			controller.pods[pname] = pod

		}
		_, cExists = controller.podDepReq[pname]
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

// evalauate if the current deployment satisfies the pod requirement.
// Make a list of pods which need to be rescheduled
// For all the pods that the controller knows of, find the bw used by the pod and the bw available to the pod
// Check if the node has sufficient bw to all dependees of the pod
func (controller *Controller) EvaluateDeployment() {
	for srcPod, podDeps := range controller.podDepReq {
		for dstPod, podReq := range podDeps {
			podActual, exists := controller.podDepActual[srcPod][dstPod]
			if exists {
				fmt.Printf("Pod dependency src = %s dst = %s bw req = %f actual = %f\n", srcPod, dstPod, podReq.bandwidth, podActual.bandwidth)
			}
		}
	}
}
