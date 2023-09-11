package main

import (
	"fmt"
	"time"
)

func computeIndegrees(podDeps map[string]map[string]bool) map[string]int {
	indegrees := make(map[string]int, 0)
	logger(fmt.Sprintf("got %d nodes", len(podDeps)))
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

func getUnvisitedVertexIdx(visited map[string]bool, topoOrder []string) int {
	for idx, _ := range topoOrder {
		if visited[topoOrder[idx]] == false {
			return idx
		}
	}
	return -1
}

func topoSortWithChain(podDeps map[string]map[string]bool) []string {
	topoOrder := topoSort(podDeps)

	visited := make(map[string]bool, 0)
	visitedGraph := make(map[string]map[string]bool, 0)

	for src, deps := range podDeps {
		visited[src] = false
		visitedGraph[src] = make(map[string]bool, 0)
		for dst, v := range deps {
			if v == true {
				visitedGraph[src][dst] = false
			}
		}
	}
	order := make([]string, 0)

	for {
		idx := getUnvisitedVertexIdx(visited, topoOrder)
		if len(order) == len(podDeps) || idx == -1 {
			break
		}
		startNode := topoOrder[idx]
		logger("cur node is " + startNode)
		lengthTo := make(map[string]int, 0)
		path := make(map[string]string, 0)
		for dst, _ := range podDeps {
			lengthTo[dst] = 0
		}
		q := make([]string, 0)
		curNode := startNode
		qVisited := make(map[string]bool, 0)
		q = append(q, curNode)
		for {
			if q == nil || len(q) == 0 {
				break
			}
			curNode = q[0]
			qVisited[curNode] = true
			if len(q) > 1 {
				q = q[1:]
			} else {
				q = q[:0]
			}
			
			logger(fmt.Sprintf("qlen= %d cur node = %s , has %d deps", len(q), curNode, len(podDeps[curNode])))
			for k, _ := range podDeps {
				if qVisited[k] == true || visited[k] == true{
					continue
				}
				if visitedGraph[k][curNode] == true {
					continue
				}
				_, exists := podDeps[k][curNode]
				if !exists {
					continue
				}
				q = append(q, k)
				logger(fmt.Sprintf("edge %s -> %s\n", k, curNode))
				if lengthTo[k] <= lengthTo[curNode]+1 {
					lengthTo[k] = lengthTo[curNode] + 1
					path[k] = curNode
					logger(fmt.Sprintf("path to %s is %s\n", k, curNode ))
				}
			}
		}
		pathLen := 0
		lastVertex := startNode
		for k, v := range lengthTo {
			logger(fmt.Sprintf("plen from %s to %s = %d\n", startNode, k, v))

			if v > pathLen {
				pathLen = lengthTo[k]
				lastVertex = k
			}
		}
		logger(fmt.Sprintf("last vertex is %s, path== %s ", lastVertex, path[lastVertex]))
		curOrder := make([]string, 0)
		curLen := 0
		visited[startNode] = true
		for {
			curVertex := lastVertex
			// path traversed in reverse
			nextVertex, exists := path[lastVertex]
			logger(fmt.Sprintf("v = %s next = %s, exits=%v\n", curVertex, nextVertex, exists))
			if exists {
				visitedGraph[lastVertex][curVertex] = true
				visited[lastVertex] = true
			}
			curLen += 1
			curOrder = append([]string{curVertex}, curOrder...)
			if curVertex == startNode {
				break
			}
			time.Sleep(10)
			//logger(fmt.Sprintf("have %d in order\n", len(order)))
			lastVertex = nextVertex
		}
		order = append(order, curOrder...)
	}
	for _, n := range order {
		logger("chain order " + n)
	}
	return order
}
func topoSort(podDeps map[string]map[string]bool) []string {
	indegrees := computeIndegrees(podDeps)
	zeroIndegreeNodes := findZeroIndegrees(indegrees)

	topoSortOrder := make([]string, 0)

	for {
		logger(fmt.Sprintf("Have %d nodes with zero indegree", len(zeroIndegreeNodes)))
		if zeroIndegreeNodes == nil || len(zeroIndegreeNodes) <= 0 {
			break
		}
		curNode := zeroIndegreeNodes[0]
		logger("cur node is " + curNode)
		topoSortOrder = append(topoSortOrder, curNode)
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
					logger(fmt.Sprintf("src = %s dst = %s indeg=%d val=%d", src, dst, indegrees[src], val))

					val, _ = indegrees[src]
					if val == 0 {
						if !find(src, topoSortOrder) && !find(src, zeroIndegreeNodes) {
							zeroIndegreeNodes = append(zeroIndegreeNodes, src)
						}
					}
				}
			}
		}

	}
	for _, node := range topoSortOrder {
		logger("topo order is " + fmt.Sprintf(" %s ", node))
	}
	return topoSortOrder
}
