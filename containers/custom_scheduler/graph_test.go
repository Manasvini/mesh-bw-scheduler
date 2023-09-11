package main

import (
	"strconv"
	"testing"
)

func getSimpleTopo(n int) map[string]map[string]bool {
	topo := make(map[string]map[string]bool, 0)

	for i := 0; i < n; i++ {
		podId := "pod_" + strconv.Itoa(i)
		topo[podId] = make(map[string]bool, 0)
		if i < n-1 {
			depPodId := "pod_" + strconv.Itoa(i+1)

			topo[podId][depPodId] = true
		}
	}
	return topo
}

func TestPodDeps(t *testing.T) {
	topo := getSimpleTopo(2)
	topoOrder := topoSort(topo)
	if len(topoOrder) != len(topo) {
		t.Fatalf("Got %d topo sorted, want %d instead", len(topoOrder), len(topo))
	}
}
func TestPodDeps1(t *testing.T) {
	topo := getSimpleTopo(3)
	topoOrder := topoSort(topo)
	if len(topoOrder) != len(topo) {
		t.Fatalf("Got %d topo sorted, want %d instead", len(topoOrder), len(topo))
	}
	expectedOrder := []string{"pod_2", "pod_1", "pod_0"}
	for i := 0; i < len(expectedOrder); i++ {
		if expectedOrder[i] != topoOrder[i] {
			t.Fatalf("got %s want %s at position %d", topoOrder[i], expectedOrder[i], i)
		}
	}
}

func TestPodDeps2(t *testing.T) {
	topo := getSimpleTopo(3)
	topo["pod_4"] = make(map[string]bool, 0)
	topo["pod_4"]["pod_2"] = true
	topoOrder := topoSort(topo)
	if len(topoOrder) != len(topo) {
		t.Fatalf("Got %d topo sorted, want %d instead", len(topoOrder), len(topo))
	}
	//expectedOrder := []string{"pod_2", "pod_1", "pod_0", "pod_4"}
	//for i := 0; i < len(expectedOrder)-2; i++ {
	//	if expectedOrder[i] != topoOrder[i] {
	//		t.Fatalf("got %s want %s at position %d", topoOrder[i], expectedOrder[i], i)
	//	}
	//}
}

func TestPodDeps3(t *testing.T) {
	topo := getSimpleTopo(3)
	topo["pod_3"] = make(map[string]bool, 0)
	topo["pod_3"]["pod_2"] = true
	topo["pod_4"] = make(map[string]bool, 0)
	topo["pod_4"]["pod_3"] = true
	topoOrder := topoSort(topo)
	chainOrder := topoSortWithChain(topo)
	if len(chainOrder) != len(topo) {
		t.Fatalf("Got %d chain topo sorted, want %d instead", len(topoOrder), len(topo))
	}
	expectedOrder := []string{"pod_2", "pod_1", "pod_0", "pod_3", "pod_4"}
	for i := 0; i < len(expectedOrder); i++ {
		if expectedOrder[i] != topoOrder[i] {
			t.Fatalf("got %s want %s at position %d", topoOrder[i], expectedOrder[i], i)
		}
	}

}
