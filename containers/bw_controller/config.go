package main

type Config struct {
	NetmonAddrs        []string
	PromAddr           string
	PromMetrics        []string
	KubeProxyAddr      string
	KubeNodesEndpoint  string
	KubePodsEndpoint   string
	KubeDeleteEndpoint string
	KubeNamespaces     []string
	MonDurationSeconds int
}
