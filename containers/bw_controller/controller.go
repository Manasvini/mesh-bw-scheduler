package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const SND_BW = "istio_tcp_sent_bytes_total"
const RCV_BW = "istio_tcp_received_bytes_total"

type Controller struct {
	promClient api.Client
	pods       PodSet
	podDeps    PodDeps
	nodes      []string
	links      LinkSet
	paths      PathSet
	metrics    []string
}

func NewController(prometheus string, metrics []string) *Controller {
	client, err := api.NewClient(api.Config{Address: prometheus})
	if err != nil {
		log.Fatalln("Error connect to the prometheus: ", err)
	}
	controller := &Controller{promClient: client, metrics: metrics}
	return controller
}

func (controller *Controller) UpdatePodMetrics() {
	for _, metric := range controller.metrics {
		controller.UpdatePrometheusMetric(metric)
	}
}
func (controller *Controller) UpdatePrometheusMetric(metric string) {
	fmt.Println(metric)
	v1api := v1.NewAPI(controller.promClient)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, warnings, err := v1api.Query(ctx, metric, time.Now())

	if err != nil {
		log.Printf("Error querying Prometheus: %s\n", err)
	}
	if len(warnings) > 0 {
		log.Printf("Warnings: %s", warnings)
	}
	fmt.Println(result)
	switch result.Type() {
	case model.ValNone:
		fmt.Println("<ValNone>")
	case model.ValScalar:
		fmt.Println("scalar")
	case model.ValVector:
		var v model.Vector
		v = result.(model.Vector)
		fmt.Println("vector")
		fmt.Println(result)
		for _, value := range v {
			src, srcExist := value.Metric["source_canonical_service"]
			dst, dstExist := value.Metric["destination_canonical_service"]
			fmt.Printf("Have %d keys\n", len(value.Metric))
			if srcExist && dstExist {
				fmt.Printf("metric = %s src = %s dest = %s metric value = %s\n", metric, src, dst, value.Value)
				srcStr := string(src)
				dstStr := string(dst)
				if metric == SND_BW {
					srcPod, srcExist := controller.pods[srcStr]
					dstPod, dstExist := controller.pods[dstStr]

					if srcExist && dstExist {
						srcpname, dstpname := srcPod.podId, dstPod.podId

						depPods, exist := controller.podDeps[srcpname]
						if exist {
							depReq, reqExist := depPods[dstpname]
							if reqExist {
								depReq.bandwidth = float64(value.Value)
								depPods[dstpname] = depReq
								controller.podDeps[srcpname] = depPods
							}
						}
					}
				} else if metric == RCV_BW {
					srcPod, srcExist := controller.pods[srcStr]
					dstPod, dstExist := controller.pods[dstStr]

					if srcExist && dstExist {
						srcpname, dstpname := srcPod.podId, dstPod.podId
						depPods, exist := controller.podDeps[dstpname]
						if exist {
							depReq, reqExist := depPods[srcpname]
							if reqExist {
								depReq.bandwidth = float64(value.Value)
								depPods[srcpname] = depReq
								controller.podDeps[dstpname] = depPods
							}
						}
					}
				}
			} else {
				for tag, val := range value.Metric {
					fmt.Printf("metric = %s key = %s val = %s metric val = %s\n", metric, tag, val, value.Value)
				}
			}
		}
	case model.ValMatrix:
		fmt.Println("matrix")
	case model.ValString:
		fmt.Println("string")
	}
}
