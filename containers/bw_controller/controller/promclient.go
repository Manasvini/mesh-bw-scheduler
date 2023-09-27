package bw_controller

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const TIMEOUT = time.Second * 30
const RCV_BW = "istio_tcp_received_bytes_total"
const SND_BW = "istio_tcp_sent_bytes_total"

type PromClient struct {
	address    string
	promClient api.Client
	metrics    []string
}

func NewPrometheusClient(address string, metrics []string) *PromClient {
	promClient, err := api.NewClient(api.Config{Address: address})
	if err != nil {
		log.Fatalln("Error connecting to Prometheus ", err)
	}
	client := &PromClient{address: address, promClient: promClient, metrics: metrics}
	return client
}

func (client *PromClient) GetPodMetrics() (PodSet, PodDeps) {

	pods := make(PodSet, 0)
	podDeps := make(PodDeps, 0)
	for _, metric := range client.metrics {
		curPods, curPodDeps := client.updateMetric(metric)
		for podName, curPod := range curPods {
			pods[podName] = curPod
			_, exists := podDeps[podName]
			if !exists {
				podDeps[podName] = make(map[string]PodDependency, 0)
			}
		}
		for src, depPods := range curPodDeps {
			for dst, dep := range depPods {
				podDeps[src][dst] = dep
			}
		}
	}
	for src, deps := range podDeps {
		for dst, dep := range deps {
			logger(fmt.Sprintf("Got dep %s -> %s bw = %f\n", src, dst, dep.Bandwidth))
		}
	}
	logger(fmt.Sprintf("Got %d pods", len(pods)))
	return pods, podDeps

}

func (client *PromClient) updateMetric(metric string) (PodSet, PodDeps) {
	logger(metric)
	pods := make(PodSet, 0)
	podDeps := make(PodDeps, 0)
	v1api := v1.NewAPI(client.promClient)
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()
	query := "rate(" + metric + "[1m])"
	result, warnings, err := v1api.Query(ctx, query, time.Now())

	if err != nil {
		log.Printf("Error querying Prometheus: %s\n", err)
	}
	if len(warnings) > 0 {
		log.Printf("Warnings: %s", warnings)
	}
	switch result.Type() {
	case model.ValNone:
		logger("<ValNone>")
	case model.ValScalar:
		logger("scalar")
	case model.ValMatrix:
		logger("matrix")
	case model.ValString:
		logger("string")
	case model.ValVector:
		var v model.Vector
		v = result.(model.Vector)
		logger("vector")
		for _, value := range v {
			src, srcExist := value.Metric["source_canonical_service"]
			dst, dstExist := value.Metric["destination_canonical_service"]
			if srcExist && dstExist {
				//fmt.Printf("metric = %s src = %s dest = %s metric value = %s\n", metric, src, dst, value.Value)
				srcStr := string(src)
				dstStr := string(dst)
				_, srcExist := pods[srcStr]
				_, dstExist := pods[dstStr]
				if !srcExist {
					pods[srcStr] = Pod{podName: srcStr}
					podDeps[srcStr] = make(map[string]PodDependency, 0)
				}
				if !dstExist {
					pods[dstStr] = Pod{podName: dstStr}
					podDeps[dstStr] = make(map[string]PodDependency, 0)
				}
				if metric == SND_BW {
					srcpname, dstpname := srcStr, dstStr
					depPods, _ := podDeps[srcpname]
					depReq, depExist := depPods[dstpname]
					if !depExist {
						depReq = PodDependency{Source: srcStr, Destination: dstStr, Bandwidth: 0, Latency: 0}
					}
					depReq.Bandwidth = float64(value.Value)
					depPods[dstpname] = depReq
					podDeps[srcpname] = depPods

					logger(fmt.Sprintf("Got metric pod %s -> %s snd bw = %f", srcpname, dstpname, depReq.Bandwidth))
				} else if metric == RCV_BW {
					srcpname, dstpname := srcStr, dstStr
					depPods, _ := podDeps[dstpname]
					depReq, depExist := depPods[srcpname]
					if !depExist {
						depReq = PodDependency{Source: dstStr, Destination: srcStr, Bandwidth: 0, Latency: 0}
					}
					depReq.Bandwidth = float64(value.Value)
					depPods[srcpname] = depReq
					podDeps[dstpname] = depPods
					logger(fmt.Sprintf("Got metric pod %s -> %s recv bw = %f", srcpname, dstpname, depReq.Bandwidth))
				}
			} else {
				for tag, val := range value.Metric {
					logger(fmt.Sprintf("metric = %s key = %s val = %s metric val = %s\n", metric, tag, val, value.Value))
				}

			}
		}
	}
	return pods, podDeps
}
