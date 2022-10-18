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

var (
	apiHost           = "0.0.0.0:8001"
	bindingsEndpoint  = "/api/v1/namespaces/epl/pods/%s/binding/"
	eventsEndpoint    = "/api/v1/watch/namespaces/epl/events"
	nodesEndpoint     = "/api/v1/nodes"
	podsEndpoint      = "/api/v1/namespaces/epl/pods/"
	watchPodsEndpoint = "/api/v1/watch/namespaces/epl/pods"
	configEndpoint	  = "/apis/apps/v1/namespaces/epl/deployments/epl-scheduler"
)

func postEvent(event Event) error {
	var b []byte
	body := bytes.NewBuffer(b)
	err := json.NewEncoder(body).Encode(event)
	if err != nil {
		return err
	}

	request := &http.Request{
		Body:          ioutil.NopCloser(body),
		ContentLength: int64(body.Len()),
		Header:        make(http.Header),
		Method:        http.MethodPost,
		URL: &url.URL{
			Host:   apiHost,
			Path:   eventsEndpoint,
			Scheme: "http",
		},
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	if resp.StatusCode != 201 {
		return errors.New("Event: Unexpected HTTP status code" + resp.Status)
	}
	return nil
}

func getNodes() (*NodeList, error) {
	var nodeList NodeList

	request := &http.Request{
		Header: make(http.Header),
		Method: http.MethodGet,
		URL: &url.URL{
			Host:   apiHost,
			Path:   nodesEndpoint,
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

	return &nodeList, nil
}

func watchUnscheduledPods() (<-chan Pod, <-chan error) {
	pods := make(chan Pod)
	errc := make(chan error, 1)

	v := url.Values{}
	v.Set("fieldSelector", "spec.nodeName=")

	request := &http.Request{
		Header: make(http.Header),
		Method: http.MethodGet,
		URL: &url.URL{
			Host:     apiHost,
			Path:     watchPodsEndpoint,
			RawQuery: v.Encode(),
			Scheme:   "http",
		},
	}
	request.Header.Set("Accept", "application/json, */*")

	go func() {
		for {
			resp, err := http.DefaultClient.Do(request)
			if err != nil {
				errc <- err
				time.Sleep(5 * time.Second)
				continue
			}

			if resp.StatusCode != 200 {
				errc <- errors.New("Invalid status code: " + resp.Status)
				time.Sleep(5 * time.Second)
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
	}()

	return pods, errc
}

func getUnscheduledPods() ([]*Pod, error) {
	var podList PodList
	unscheduledPods := make([]*Pod, 0)

	v := url.Values{}
	v.Set("fieldSelector", "spec.nodeName=")

	request := &http.Request{
		Header: make(http.Header),
		Method: http.MethodGet,
		URL: &url.URL{
			Host:     apiHost,
			Path:     podsEndpoint,
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

func getPods() (*PodList, error) {
	var podList PodList

	v := url.Values{}
	v.Add("fieldSelector", "status.phase=Running")
	v.Add("fieldSelector", "status.phase=Pending")

	request := &http.Request{
		Header: make(http.Header),
		Method: http.MethodGet,
		URL: &url.URL{
			Host:     apiHost,
			Path:     podsEndpoint,
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

func getConfig() (string, error) {
	var deployment Deployment

	request := &http.Request{
		Header: make(http.Header),
		Method: http.MethodGet,
		URL: &url.URL{
			Host:     apiHost,
			Path:     configEndpoint,
			Scheme:   "http",
		},
	}
	request.Header.Set("Accept", "application/json, */*")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	err = json.NewDecoder(resp.Body).Decode(&deployment)
	if err != nil {
		return "", err
	}

	if val, ok := deployment.Metadata.Annotations["epl/staticinstance"]; ok {
		return val, nil
	}

	return "", nil
}

func fit(pod *Pod) ([]Node, error) {
	nodeList, err := getNodes()
	if err != nil {
		return nil, err
	}

	config, err := getConfig()
	if err != nil {
		return nil, err
	}

	var nodes []Node

	for _, node := range nodeList.Items {
		if node.Metadata.Name == config {
			nodes = append(nodes, node)
		}
	}

	if (len(nodes) == 0) || (config == "") {
		// Emit a Kubernetes event that the Pod failed to be scheduled.
		timestamp := time.Now().UTC().Format(time.RFC3339)
		event := Event{
			Count:          1,
			Message:        fmt.Sprintf("pod (%s) failed to fit in any node\n", pod.Metadata.Name),
			Reason:         "FailedScheduling",
			LastTimestamp:  timestamp,
			FirstTimestamp: timestamp,
			Type:           "Warning",
			Source:         EventSource{Component: "epl-scheduler"},
			InvolvedObject: ObjectReference{
				Kind:      "Pod",
				Name:      pod.Metadata.Name,
				Namespace: "epl",
				Uid:       pod.Metadata.Uid,
			},
		}

		postEvent(event)
	}

	return nodes, nil
}

func bind(pod *Pod, node Node) error {
	binding := Binding{
		ApiVersion: "v1",
		Kind:       "Binding",
		Metadata:   Metadata{Name: pod.Metadata.Name},
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
		return err
	}

	request := &http.Request{
		Body:          ioutil.NopCloser(body),
		ContentLength: int64(body.Len()),
		Header:        make(http.Header),
		Method:        http.MethodPost,
		URL: &url.URL{
			Host:   apiHost,
			Path:   fmt.Sprintf(bindingsEndpoint, pod.Metadata.Name),
			Scheme: "http",
		},
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	if resp.StatusCode != 201 {
		return errors.New("Binding: Unexpected HTTP status code" + resp.Status)
	}

	// Emit a Kubernetes event that the Pod was scheduled successfully.
	message := fmt.Sprintf("Successfully assigned %s to %s", pod.Metadata.Name, node.Metadata.Name)
	timestamp := time.Now().UTC().Format(time.RFC3339)
	event := Event{
		Count:          1,
		Message:        message,
		Reason:         "Scheduled",
		LastTimestamp:  timestamp,
		FirstTimestamp: timestamp,
		Type:           "Normal",
		Source:         EventSource{Component: "hightower-scheduler"},
		InvolvedObject: ObjectReference{
			Kind:      "Pod",
			Name:      pod.Metadata.Name,
			Namespace: "default",
			Uid:       pod.Metadata.Uid,
		},
	}
	logger(message)
	return postEvent(event)
}
