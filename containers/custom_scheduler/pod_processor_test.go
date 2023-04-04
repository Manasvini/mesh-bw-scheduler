package main

import (
	"testing"
)

func getPodSimpleTopo() map[string]Pod {
	pods := make(map[string]Pod, 0)

	// p0 --> p1 --> p2
	ann1 := map[string]string{"dependee/bw/pod_1": "1Mbps", "dependee/latency/pod_1": "10ms"}
	ann2 := map[string]string{"dependee/bw/pod_2": "1Mbps", "dependee/latency/pod_2": "10ms", "depender/bw/pod_0": "1Mbps", "depender/latency/pod_0": "10ms"}
	ann3 := map[string]string{"depender/bw/pod_1": "1Mbps", "depender/latency/pod_1": "10ms"}

	podMeta := Metadata{Name: "pod_0", Annotations: ann1}
	pods["pod_0"] = Pod{Kind: "pod", Metadata: podMeta}
	podMeta = Metadata{Name: "pod_1", Annotations: ann2}
	pods["pod_1"] = Pod{Kind: "pod", Metadata: podMeta}
	podMeta = Metadata{Name: "pod_2", Annotations: ann3}
	pods["pod_2"] = Pod{Kind: "pod", Metadata: podMeta}

	return pods
}
func getPodSimpleTopoIncomplete() map[string]Pod {
	pods := make(map[string]Pod, 0)

	// p0 --> p1 --> p2
	ann1 := map[string]string{"dependee/bw/pod_1": "1Mbps", "dependee/latency/pod_1": "10ms"}
	ann2 := map[string]string{"dependee/bw/pod_2": "1Mbps", "dependee/latency/pod_2": "10ms", "depender/bw/pod_0": "1Mbps", "depender/latency/pod_0": "10ms"}
	//ann3 := map[string]string{"depender/bw/pod_1": "1Mbps", "depender/latency/pod_1": "10ms"}

	podMeta := Metadata{Name: "pod_0", Annotations: ann1}
	pods["pod_0"] = Pod{Kind: "pod", Metadata: podMeta}
	podMeta = Metadata{Name: "pod_1", Annotations: ann2}
	pods["pod_1"] = Pod{Kind: "pod", Metadata: podMeta}

	return pods
}

func getPodDisconnectedTopo() map[string]Pod {
	pods := make(map[string]Pod, 0)

	// p0 --> p1 --> p2 p4 --> p5
	ann1 := map[string]string{"dependee/bw/pod_1": "1Mbps", "dependee/latency/pod_1": "10ms"}
	ann2 := map[string]string{"dependee/bw/pod_2": "1Mbps", "dependee/latency/pod_2": "10ms", "depender/bw/pod_0": "1Mbps", "depender/latency/pod_0": "10ms"}
	ann3 := map[string]string{"depender/bw/pod_1": "1Mbps", "depender/latency/pod_1": "10ms"}

	ann4 := map[string]string{"dependee/bw/pod_5": "1Mbps", "dependee/latency/pod_5": "10ms"}
	ann5 := map[string]string{"depender/bw/pod_4": "1Mbps", "depender/latency/pod_5": "10ms"}

	podMeta := Metadata{Name: "pod_0", Annotations: ann1}
	pods["pod_0"] = Pod{Kind: "pod", Metadata: podMeta}
	podMeta = Metadata{Name: "pod_1", Annotations: ann2}
	pods["pod_1"] = Pod{Kind: "pod", Metadata: podMeta}
	podMeta = Metadata{Name: "pod_2", Annotations: ann3}
	pods["pod_2"] = Pod{Kind: "pod", Metadata: podMeta}
	podMeta = Metadata{Name: "pod_4", Annotations: ann4}
	pods["pod_4"] = Pod{Kind: "pod", Metadata: podMeta}
	podMeta = Metadata{Name: "pod_5", Annotations: ann5}
	pods["pod_5"] = Pod{Kind: "pod", Metadata: podMeta}

	return pods
}

func TestIncompletePodGraph1(t *testing.T) {
	pods := getPodSimpleTopoIncomplete()
	pp := NewPodProcessor()
	pp.unscheduledPods = pods
	podGraph := pp.GetPodGraph()

	if nil != podGraph {
		t.Fatalf("Want empty pod graph got %d instead", len(podGraph))
	}
}

func TestGetPodGraph2(t *testing.T) {
	pods := getPodSimpleTopo()
	pp := NewPodProcessor()
	pp.unscheduledPods = pods
	podGraph := pp.GetPodGraph()

	wantPodGraph := map[string]map[string]bool{"pod_0": {"pod_1": true}, "pod_1": {"pod_0": true, "pod_2": true}, "pod_2": {"pod_1": true}}

	if len(wantPodGraph) != len(podGraph) {
		t.Fatalf("Want %d got %d instead", len(wantPodGraph), len(podGraph))
	}
}

func TestGetPodGraphComponents(t *testing.T) {
	pods := getPodSimpleTopo()
	pp := NewPodProcessor()
	pp.unscheduledPods = pods
	podGraph := pp.GetPodGraph()

	podgroups := pp.GetPodGroups(podGraph)
	if len(podgroups) != 1 {
		t.Fatalf("Want 1 pod group, got %d instead", len(podgroups))
	}
}

func TestGetPodGraphComponents1(t *testing.T) {
	pods := getPodDisconnectedTopo()
	pp := NewPodProcessor()
	pp.unscheduledPods = pods
	podGraph := pp.GetPodGraph()

	podgroups := pp.GetPodGroups(podGraph)
	if len(podgroups) != 2 {
		t.Fatalf("Want 2 pod group, got %d instead", len(podgroups))
	}
}

func TestDepGraphComponents(t *testing.T) {
	pods := getPodSimpleTopo()
	pp := NewPodProcessor()
	pp.unscheduledPods = pods
	podGraph := pp.GetPodGraph()

	podgroups := pp.GetPodGroups(podGraph)

	wantPodGraph := map[string]map[string]bool{"pod_0": {"pod_1": true}, "pod_1": {"pod_2": true}}

	podList := make([]Pod, 0)
	for _, podName := range podgroups[0] {
		podList = append(podList, pods[podName])
	}

	depGraph := pp.GetPodDependencyGraph(podList)

	if len(wantPodGraph) != len(depGraph) {
		t.Fatalf("Want %d got %d instead", len(wantPodGraph), len(depGraph))
	}
}
