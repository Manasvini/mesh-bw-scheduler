package bw_controller

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

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

type KubeClient struct {
	address       string
	namespaces    []string
	nodesEndpoint string
	podsEndpoint  string
}

func NewKubeClient(address string, nodesEndpoint string, podsEndpoint string, namespaces []string) *KubeClient {
	client := &KubeClient{address: address, nodesEndpoint: nodesEndpoint, podsEndpoint: podsEndpoint, namespaces: namespaces}
	success := client.WaitForProxy()
	if !success {
		panic("Unable to connect to K3s proxy")
	}
	return client

}

func (client *KubeClient) WaitForProxy() bool {
	logger("Waiting for proxy to start")

	for i := 1; i < 5; i++ {
		time.Sleep(time.Second)
		request := &http.Request{
			Header: make(http.Header),
			Method: http.MethodGet,
			URL: &url.URL{
				Host:   client.address,
				Path:   "",
				Scheme: "http",
			},
		}
		_, err := http.DefaultClient.Do(request)
		if err != nil {
			continue
		}
		return true
	}
	return false
}

func (client *KubeClient) GetNodes() (*NodeList, error) {
	var nodeList NodeList

	request := &http.Request{
		Header: make(http.Header),
		Method: http.MethodGet,
		URL: &url.URL{
			Host:   client.address,
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

	return &nodeList, nil
}

func (client *KubeClient) GetPods() []PodList {
	podLists := make([]PodList, 0)
	for _, ns := range client.namespaces {
		nsPodList, err := client.GetPodsInNamespace(ns)
		if err != nil {
			log.Fatalln("kube proxy returned error")
		}
		podLists = append(podLists, *nsPodList)

	}
	return podLists
}

func (client *KubeClient) GetPodsInNamespace(ns string) (*PodList, error) {
	var podList PodList

	v := url.Values{}
	v.Add("fieldSelector", "status.phase=Running")
	//v.Add("fieldSelector", "status.phase=Pending")

	request := &http.Request{
		Header: make(http.Header),
		Method: http.MethodGet,
		URL: &url.URL{
			Host:     client.address,
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
