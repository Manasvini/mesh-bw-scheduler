package main

import (
	"time"
)

var (
	prometheus string        = "http://0.0.0.0:9090/"
	timeout    time.Duration = time.Second * 30
)

func main() {
	metrics := []string{"istio_request_duration_milliseconds", "istio_tcp_sent_bytes_total", "istio_tcp_received_bytes_total"}
	controller := NewController(prometheus, metrics)
	controller.UpdatePodMetrics()
}
