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
	"fmt"
	"sync"
	"time"
)

var processorLock = &sync.Mutex{}

func reconcileUnscheduledPods(interval int, done chan struct{}, wg *sync.WaitGroup) {
	for {
		select {
		case <-time.After(time.Duration(interval) * time.Second):
			err := schedulePods()
			if err != nil {
				logger(err)
			}
		case <-done:
			wg.Done()
			logger("Stopped reconciliation loop.")
			return
		}
	}
}

func monitorUnscheduledPods(done chan struct{}, wg *sync.WaitGroup) {
	pods, errc := watchUnscheduledPods()

	for {
		select {
		case err := <-errc:
			logger(err)
		case pod := <-pods:
			processorLock.Lock()
			time.Sleep(2 * time.Second)
			err := schedulePod(&pod)
			if err != nil {
				logger(err)
			}
			processorLock.Unlock()
		case <-done:
			wg.Done()
			logger("Stopped scheduler.")
			return
		}
	}
}

func schedulePod(pod *Pod) error {
	nodes, err := fit(pod)
	if err != nil {
		logger(err)
		return err
	}
	if len(nodes) == 0 {
		return fmt.Errorf("Unable to schedule pod (%s) failed to fit in any node", pod.Metadata.Name)
	}
	err = bind(pod, nodes[0])
	if err != nil {
		logger(err)
		return err
	}
	return nil
}

func schedulePods() error {
	processorLock.Lock()
	defer processorLock.Unlock()
	pods, err := getUnscheduledPods()
	if err != nil {
		logger(err)
		return err
	}
	logger(fmt.Sprintf("got %d pods", len(pods)))
	for _, pod := range pods {
		err := schedulePod(pod)
		if err != nil {
			logger(err)
		}
	}
	return nil
}
