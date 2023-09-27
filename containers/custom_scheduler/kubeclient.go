// Copyright 2016 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type KubeClient struct {
	apiHost           string
	bindingsEndpoint  string
	eventsEndpoint    string
	nodesEndpoint     string
	watchPodsEndpoint string
	configEndpoint    string
	podsEndpoint      string
	namespaces        []string
	metricsEndpoint   string
}

func (client *KubeClient) WaitForProxy() int {
	logger("Waiting for proxy to start")

	for i := 1; i < 5; i++ {
		time.Sleep(time.Second)

		request := &http.Request{
			Header: make(http.Header),
			Method: http.MethodGet,
			URL: &url.URL{
				Host:   client.apiHost,
				Path:   "",
				Scheme: "http",
			},
		}

		_, err := http.DefaultClient.Do(request)
		if err != nil {
			continue
		}

		return 1
	}

	return 0
}

func (client *KubeClient) PostEvent(event Event, ns string) error {
	var b []byte
	body := bytes.NewBuffer(b)
	err := json.NewEncoder(body).Encode(event)
	if err != nil {
		logger(err)
		return err
	}

	request := &http.Request{
		Body:          ioutil.NopCloser(body),
		ContentLength: int64(body.Len()),
		Header:        make(http.Header),
		Method:        http.MethodPost,
		URL: &url.URL{
			Host:   client.apiHost,
			Path:   fmt.Sprintf(client.eventsEndpoint, ns),
			Scheme: "http",
		},
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		logger(err)
		return err
	}
	if resp.StatusCode != 201 {
		logger(errors.New("Event: Unexpected HTTP status code" + resp.Status))
		return errors.New("Event: Unexpected HTTP status code" + resp.Status)
	}
	return nil
}

func (client *KubeClient) GetNodeMetrics() (*NodeMetricsList, error) {
	var nodeMetricsList NodeMetricsList

	request := &http.Request{
		Header: make(http.Header),
		Method: http.MethodGet,
		URL: &url.URL{
			Host:   client.apiHost,
			Path:   client.metricsEndpoint,
			Scheme: "http",
		},
	}
	request.Header.Set("Accept", "application/json, */*")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	err = json.NewDecoder(resp.Body).Decode(&nodeMetricsList)
	if err != nil {
		return nil, err
	}

	return &nodeMetricsList, nil
}

func (client *KubeClient) GetNodes() (*NodeList, error) {
	var nodeList NodeList

	request := &http.Request{
		Header: make(http.Header),
		Method: http.MethodGet,
		URL: &url.URL{
			Host:   client.apiHost,
			Path:   client.nodesEndpoint,
			Scheme: "http",
		},
	}
	request.Header.Set("Accept", "application/json, */*")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	err = json.NewDecoder(resp.Body).Decode(&nodeList)
	if err != nil {
		return nil, err
	}
	finalNodeList := make([]Node, 0)
	for _, node := range nodeList.Items {
		logger("node = " + node.Metadata.Name)
		if len(node.Spec.Taints) == 0 {
			finalNodeList = append(finalNodeList, node)
			logger("added node ")
		} else {
			add := true
			for _, taint := range node.Spec.Taints {
				if taint.Effect == "NoSchedule" {
					logger("node " + node.Metadata.Name + " cant be used")
					add = false
					break
				}
			}
			if add == true {
				finalNodeList = append(finalNodeList, node)
				logger("added node")
			}
		}
	}
	nodeList.Items = finalNodeList
	return &nodeList, nil
}

func (client *KubeClient) WatchUnscheduledPods() (<-chan Pod, <-chan error) {
	pods := make(chan Pod)
	errc := make(chan error, 1)

	v := url.Values{}
	v.Set("fieldSelector", "status.phase==Pending")

	//request := &http.Request{
	//	Header: make(http.Header),
	//	Method: http.MethodGet,
	//	URL: &url.URL{
	//		Host:     client.apiHost,
	//		Path:     client.watchPodsEndpoint,
	//		RawQuery: v.Encode(),
	//		Scheme:   "http",
	//	},
	//}
	//request.Header.Set("Accept", "application/json, */*")
	for _, ns := range client.namespaces {
		go func(ns string) {
			for {
				logger("ns = " + ns)
				request := &http.Request{
					Header: make(http.Header),
					Method: http.MethodGet,
					URL: &url.URL{
						Host:     client.apiHost,
						Path:     fmt.Sprintf(client.watchPodsEndpoint, ns),
						RawQuery: v.Encode(),
						Scheme:   "http",
					},
				}
				resp, err := http.DefaultClient.Do(request)
				if err != nil {
					errc <- err
					//time.Sleep(5 * time.Second)
					continue
				}

				if resp.StatusCode != 200 {
					errc <- errors.New("Invalid status code: " + resp.Status)
					//time.Sleep(5 * time.Second)
					continue
				}

				decoder := json.NewDecoder(resp.Body)
				for {
					var event PodWatchEvent
					err = decoder.Decode(&event)
					if err != nil {
						errc <- err
						break
					}

					if event.Type == "ADDED" {
						pods <- event.Object
					}
				}
			}
			time.Sleep(5 * time.Second)
		}(ns)
	}

	return pods, errc
}

func (client KubeClient) GetUnscheduledPods() ([]*Pod, error) {
	pods := make([]*Pod, 0)
	for _, ns := range client.namespaces {
		logger("monitoring namespace " + ns)
		curpods, err := client.getUnscheduledPodsOne(ns)
		if err == nil {
			pods = append(pods, curpods...)
		} else {
			logger(fmt.Sprintf("Got error %v", err))
		}
	}
	return pods, nil
}

func (client KubeClient) getUnscheduledPodsOne(ns string) ([]*Pod, error) {
	var podList PodList
	unscheduledPods := make([]*Pod, 0)

	v := url.Values{}
	v.Set("fieldSelector", "spec.nodeName=")

	request := &http.Request{
		Header: make(http.Header),
		Method: http.MethodGet,
		URL: &url.URL{
			Host:     client.apiHost,
			Path:     fmt.Sprintf(client.podsEndpoint, ns),
			RawQuery: v.Encode(),
			Scheme:   "http",
		},
	}
	request.Header.Set("Accept", "application/json, */*")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return unscheduledPods, err
	}
	err = json.NewDecoder(resp.Body).Decode(&podList)
	if err != nil {
		return unscheduledPods, err
	}

	for _, pod := range podList.Items {
		if pod.Spec.SchedulerName == schedulerName {
			unscheduledPods = append(unscheduledPods, &pod)
		}
	}

	return unscheduledPods, nil
}

func (client *KubeClient) GetPods() ([]*PodList, error) {
	podLists := make([]*PodList, 0)
	for _, ns := range client.namespaces {
		plist, err := client.getPodsOne(ns)
		if err == nil {
			podLists = append(podLists, plist)
		} else {
			logger(fmt.Sprintf("Got error %v", err))
		}
	}
	return podLists, nil

}
func (client *KubeClient) getPodsOne(ns string) (*PodList, error) {
	var podList PodList

	v := url.Values{}
	request := &http.Request{
		Header: make(http.Header),
		Method: http.MethodGet,
		URL: &url.URL{
			Host:     client.apiHost,
			Path:     fmt.Sprintf(client.podsEndpoint, ns),
			RawQuery: v.Encode(),
			Scheme:   "http",
		},
	}
	request.Header.Set("Accept", "application/json, */*")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	err = json.NewDecoder(resp.Body).Decode(&podList)
	if err != nil {
		return nil, err
	}
	return &podList, nil
}

func (client *KubeClient) Bind(pod Pod, node Node) error {
	binding := Binding{
		ApiVersion: "v1",
		Kind:       "Binding",
		Metadata:   Metadata{Name: pod.Metadata.Name, Namespace: pod.Metadata.Namespace, Annotations: pod.Metadata.Annotations},
		Target: Target{
			ApiVersion: "v1",
			Kind:       "Node",
			Name:       node.Metadata.Name,
		},
	}

	var b []byte
	body := bytes.NewBuffer(b)
	err := json.NewEncoder(body).Encode(binding)
	if err != nil {
		logger(err)
		return err
	}

	request := &http.Request{
		Body:          ioutil.NopCloser(body),
		ContentLength: int64(body.Len()),
		Header:        make(http.Header),
		Method:        http.MethodPost,
		URL: &url.URL{
			Host:   apiHost,
			Path:   fmt.Sprintf(client.bindingsEndpoint, pod.Metadata.Namespace, pod.Metadata.Name),
			Scheme: "http",
		},
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		logger(err)
		return err
	}
	if resp.StatusCode != 201 {
		logger(errors.New("Binding: Unexpected HTTP status code: " + resp.Status))
		return errors.New("Binding: Unexpected HTTP status code: " + resp.Status)
	}

	message := fmt.Sprintf("Successfully assigned %s to %s", pod.Metadata.Name, node.Metadata.Name)
	logger(message)

	// // Emit a Kubernetes event that the Pod was scheduled successfully.
	// timestamp := time.Now().UTC().Format(time.RFC3339)
	// event := Event{
	// 	Count:          1,
	// 	Message:        message,
	// 	Reason:         "Scheduled",
	// 	LastTimestamp:  timestamp,
	// 	FirstTimestamp: timestamp,
	// 	Type:           "Normal",
	// 	Source:         EventSource{Component: "hightower-scheduler"},
	// 	InvolvedObject: ObjectReference{
	// 		Kind:      "Pod",
	// 		Name:      pod.Metadata.Name,
	// 		Namespace: "default",
	// 		Uid:       pod.Metadata.Uid,
	// 	},
	// }
	// return postEvent(event)

	return nil
}
