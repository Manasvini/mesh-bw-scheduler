package measurement

type NodeMetric struct {
	Name      string `json:"__name__,omitempty"`
	Container string `json:"container,omitempty"`
	Device    string `json:"device,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"`
	Instance  string `json:"instance,omitempty"`
	Job       string `json:"job,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Pod       string `json:"pod,omitempty"`
	Service   string `json:"service,omitempty"`
}

type NodeMetricValue struct {
	UnixTime float64
	Value    string
}

type NodeMeasurementRaw struct {
	Metric NodeMetric    `json:"metric,omitempty"`
	Value  []interface{} `json:"value,omitempty"`
}

type NodeMeasurement struct {
	Metric NodeMetric
	Value  NodeMetricValue
}
