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

func (client *PromClient) UpdatePodMetrics(pods *PodSet, podDeps *PodDeps) {
	for _, metric := range client.metrics {
		*pods, *podDeps = client.updateMetric(metric, *pods, *podDeps)
	}
}

func (client *PromClient) updateMetric(metric string, pods PodSet, podDeps PodDeps) (PodSet, PodDeps) {
	fmt.Println(metric)
	fmt.Printf("controller knows about %d pods\n", len(pods))
	v1api := v1.NewAPI(client.promClient)
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()

	result, warnings, err := v1api.Query(ctx, metric, time.Now())

	if err != nil {
		log.Printf("Error querying Prometheus: %s\n", err)
	}
	if len(warnings) > 0 {
		log.Printf("Warnings: %s", warnings)
	}
	switch result.Type() {
	case model.ValNone:
		fmt.Println("<ValNone>")
	case model.ValScalar:
		fmt.Println("scalar")
	case model.ValMatrix:
		fmt.Println("matrix")
	case model.ValString:
		fmt.Println("string")
	case model.ValVector:
		var v model.Vector
		v = result.(model.Vector)
		fmt.Println("vector")
		for _, value := range v {
			src, srcExist := value.Metric["source_canonical_service"]
			dst, dstExist := value.Metric["destination_canonical_service"]
			//fmt.Printf("Have %d keys\n", len(value.Metric))
			if srcExist && dstExist {
				//fmt.Printf("metric = %s src = %s dest = %s metric value = %s\n", metric, src, dst, value.Value)
				srcStr := string(src)
				dstStr := string(dst)
				if metric == SND_BW {
					srcPod, srcExist := pods[srcStr]
					dstPod, dstExist := pods[dstStr]

					if srcExist && dstExist {
						srcpname, dstpname := srcPod.podId, dstPod.podId

						depPods, exist := podDeps[srcpname]
						if exist {
							depReq, reqExist := depPods[dstpname]
							if reqExist {
								depReq.bandwidth = float64(value.Value)
								depPods[dstpname] = depReq
								podDeps[srcpname] = depPods

								logger(fmt.Sprintf("Got metric pod %s -> %s snd bw = %f", srcpname, dstpname, depReq.bandwidth))
							}
						}
					}
				} else if metric == RCV_BW {
					srcPod, srcExist := pods[srcStr]
					dstPod, dstExist := pods[dstStr]

					if srcExist && dstExist {
						srcpname, dstpname := srcPod.podId, dstPod.podId
						depPods, exist := podDeps[dstpname]
						if exist {
							depReq, reqExist := depPods[srcpname]
							if reqExist {
								depReq.bandwidth = float64(value.Value)
								depPods[srcpname] = depReq
								podDeps[dstpname] = depPods
								logger(fmt.Sprintf("Got metric pod %s -> %s recv bw = %f", srcpname, dstpname, depReq.bandwidth))
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
	}
	return pods, podDeps
}
