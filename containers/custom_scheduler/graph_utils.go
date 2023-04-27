package main

import "fmt"

func computeIndegrees(podDeps map[string]map[string]bool) map[string]int {
	indegrees := make(map[string]int, 0)

	for src, deps := range podDeps {
		logger(fmt.Sprintf(" src = %s deps = %d", src, len(deps)))
		indegrees[src] = len(deps)
	}
	return indegrees
}

func findZeroIndegrees(indegrees map[string]int) []string {
	zeroIndeg := make([]string, 0)
	for src, indeg := range indegrees {
		if indeg == 0 {
			zeroIndeg = append(zeroIndeg, src)
		}
	}
	return zeroIndeg
}

func find(val string, vals []string) bool {
	for _, v := range vals {
		if val == v {
			return true
		}
	}
	return false
}

func getNeighbors(node string, graph map[string]map[string]bool) []string {
	neighbors := make([]string, 0)
	for src, deps := range graph {
		_, exists := deps[node]
		if exists {
			neighbors = append(neighbors, src)
		}
	}
	return neighbors
}

func topoSort(podDeps map[string]map[string]bool) []string {
	indegrees := computeIndegrees(podDeps)
	zeroIndegreeNodes := findZeroIndegrees(indegrees)

	topoSortOrder := make([]string, 0)

	for {
		if zeroIndegreeNodes == nil || len(zeroIndegreeNodes) <= 0 {
			break
		}
		curNode := zeroIndegreeNodes[0]
		topoSortOrder = append(topoSortOrder, curNode)
		logger("added node " + curNode + " to topo")
		if len(zeroIndegreeNodes) > 1 {
			zeroIndegreeNodes = zeroIndegreeNodes[1:len(zeroIndegreeNodes)]
		} else {
			zeroIndegreeNodes = make([]string, 0)
		}
		for src, deps := range podDeps {
			for dst, _ := range deps {
				if dst == curNode {
					val, _ := indegrees[src]
					indegrees[src] = val - 1
				}
			}

			val, _ := indegrees[src]
			logger(fmt.Sprintf("%s has indegree %d", src, val))
			if val == 0 {
				if !find(src, topoSortOrder) && !find(src, zeroIndegreeNodes) {
					zeroIndegreeNodes = append(zeroIndegreeNodes, src)
					logger("added " + src + " to topo")
				}
			}
		}

	}
	for _, node := range topoSortOrder {
		logger("topo order is " + fmt.Sprintf(" %s ", node))
	}
	return topoSortOrder
}
