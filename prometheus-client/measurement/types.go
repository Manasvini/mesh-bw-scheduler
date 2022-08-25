package measurement

type NodeMetric struct {
	name      string `json:"__name__,omitempty"`
	container string `json:"container,omitempty"`
	device    string `json:"device,omitempty"`
	endpoint  string `json:"endpoint,omitempty"`
	instance  string `json:"instance,omitempty"`
	job       string `json:"job,omitempty"`
	namespace string `json:"namespace,omitempty"`
	pod       string `json:"pod,omitempty"`
	service   string `json:"service,omitempty"`
}

type NodeMetricValue struct {
	unixTime float64
	value    string
}

type NodeMeasurementRaw struct {
	metric NodeMetric    `json:"metric,omitempty"`
	value  []interface{} `json:"value,omitempty"`
}

type NodeMeasurement struct {
	metric NodeMetric
	value  NodeMetricValue
}
