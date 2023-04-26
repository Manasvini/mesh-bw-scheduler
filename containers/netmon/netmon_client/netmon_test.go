package netmon_client

import (
	"fmt"
	"strconv"
	"testing"
)

func GetNetInfo(maxHops int) (LinkSet, PathSet) {
	links := make(LinkSet, 0)
	paths := make(PathSet, 0)
	for i := 0; i < maxHops; i++ {
		j := i + 1
		if j == maxHops {
			break
		}
		src := strconv.Itoa(i)
		dst := strconv.Itoa(j)
		fmt.Printf("src = %s dst = %s bw = %d\n", src, dst, i+1)
		lMap, exists := links[src]
		pMap, pExists := paths[src]
		if !exists {
			lMap = make(map[string]Link, 0)
		}
		if !pExists {
			pMap = make(map[string]Path, 0)
		}
		link := Link{Source: src, Destination: dst, Bandwidth: float64(i + 1)}
		hops := []string{dst}
		path := Path{Source: src, Destination: dst, Bandwidth: float64(i + 1), Hops: hops}
		lMap[dst] = link
		pMap[dst] = path
		paths[src] = pMap
		lMap, exists = links[dst]
		pMap, pExists = paths[dst]
		if !exists {
			lMap = make(map[string]Link, 0)

		}
		if !pExists {
			pMap = make(map[string]Path, 0)
		}
		link = Link{Source: dst, Destination: src, Bandwidth: float64(i + 1)}
		lMap[src] = link
		hops = []string{src}
		path = Path{Source: dst, Destination: src, Bandwidth: float64(i + 1), Hops: hops}
		pMap[src] = path
		paths[dst] = pMap
	}
	for i := 0; i < maxHops; i++ {
		for j := 0; j < maxHops; j++ {
			src := strconv.Itoa(i)
			dst := strconv.Itoa(j)
			hops := make([]string, 0)
			start := i
			end := j
			if start == end {
				continue
			}
			fmt.Printf("start = %d end = %d\n", start, end)
			if end > start {

				for k := start; k <= end; k++ {
					h := strconv.Itoa(k)
					hops = append(hops, h)
				}
			} else {
				for k := start; k >= end; k-- {
					hops = append(hops, strconv.Itoa(k))

				}
			}
			if len(hops) == 1 {
				continue

			}
			fmt.Printf("hops = %d\n", len(hops))
			path := Path{Source: src, Destination: dst, Hops: hops}
			pMap, exists := paths[src]
			if !exists {
				pMap = make(map[string]Path, 0)
			}
			pMap[dst] = path
			paths[src] = pMap
		}
	}
	return links, paths
}

func TestUpdatePaths1(t *testing.T) {
	links, paths := GetNetInfo(2)
	dummyClient := NetmonClient{}
	dummyClient.computePathBw(links, &paths)
	if len(paths) != 2 {
		t.Fatalf("Expected 2 nodes, got %d\n", len(paths))
	}
}

func TestUpdatePaths2(t *testing.T) {
	links, paths := GetNetInfo(3)
	dummyClient := NetmonClient{}
	dummyClient.computePathBw(links, &paths)
	if len(paths) != 3 {
		t.Fatalf("Expected 3 nodes, got %d\n", len(paths))
	}
	for _, pMap := range paths {
		if len(pMap) != 2 {
			t.Fatalf("Expected 2 nodes, got %d instead\n", len(pMap))
		}
		for _, path := range pMap {
			if path.Bandwidth > 1.0 {
				t.Fatalf("Expected bottleneck bw 1, got %f instead", path.Bandwidth)
			}
		}
	}
}
