package main

type Config struct {
	NetmonAddrs       []string
	PromAddr          string
	PromMetrics       []string
	KubeProxyAddr     string
	KubeNodesEndpoint string
	KubePodsEndpoint  string
	KubeNamespaces    []string
}
