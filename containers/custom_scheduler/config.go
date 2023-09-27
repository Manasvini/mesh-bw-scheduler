package main

type Config struct {
	ApiHost           string
	BindingsEndpoint  string
	EventsEndpoint    string
	NodesEndpoint     string
	PodsEndpoint      string
	WatchPodsEndpoint string
	ConfigEndpoint    string
	MetricsEndpoint   string
	NetmonAddrs       []string
	Namespaces        []string
	PromAddr	  string
	PromMetrics       []string
}
