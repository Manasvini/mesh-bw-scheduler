package main

import "fmt"

func computeIndegrees(podDeps map[string]map[string]bool) map[string]int {
	indegrees := make(map[string]int, 0)
	logger(fmt.Sprintf("got %d nodes", len(podDeps)))
	for src, deps := range podDeps {
		//logger(fmt.Sprintf(" src = %s deps = %d", src, len(deps)))
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

func getUnvisitedVertexIdx(visited map[string]bool, topoOrder) int {
    for idx, _ := range topoOrder {
        if visited[topoOrder[idx]] == false {
            return idx
        }
    }
    return -1
}

func topoSortWithChain(podDeps map[string]map[string]bool) []string {
    topoOrder := topSort(podDeps)

    visited := make(map[string]bool, 0)
    visitedGraph := make(map[string]map[string]bool, 0)

    for src, deps := range podDeps {
        visited[src] = false
        visitedGraph[src] = make(map[string]bool, 0)
        for dst, v := range podDeps {
            if v == true {
                visitedGraph[src][dst] = false
            }
        }
    }
    order := []string
    
    for {
        idx := getUnvisitedVertexIndex(visited, topoOrder)
        if len(order) == len(podDeps) {
            break
        }
        curNode := topoOrder[idx]
        lengthTo := make(map[string]int, 0)
        path := make(map[string]string, 0)
        for _, dst := range podDeps {
            lengthTo[dst] = 0
        }
        for _, v := range topoOrder {
            if visited[v] == true {
                continue
            }
            for k, _ := range podDeps[v]{
                if visitedGraph[k][v] == true {
                    continue
                }
                if lengthTo[k] <= lengthTo[v] + 1 {
                    lengthTo[k] = lengthTo[v] + 1
                    path[k] = v
                }
            }
        }
        pathLen := 0
        lastVertex := curNode
        for k, v := range lengthTo {
            if lengthTo[k] > pathLen {
                pathLen = lengthTo[k]
                lastVertex = k 
            }
        }
        visited[curNode] = true
        for {
            curVertex := lastVertex
            // path traversed in reverse
            lastVertex, exists = path[lastVertex]
            if exists {
                visitedGraph[lastVertex][curVertex] = true
                visited[lastVertex] = true
            }
            order = append([]string{lastVertex}, order...)
            if lastVertex == curNode {
                break
            }
          
        }
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
