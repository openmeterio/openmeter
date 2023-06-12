// Copyright © 2023 Tailfin Cloud Inc.
//
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
	"context"
	"fmt"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/google/uuid"
	openmeter "github.com/openmeterio/openmeter/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// See: https://github.com/kubernetes/client-go/tree/master/examples/in-cluster-client-configuration
func main() {
	ctx := context.Background()
	om, err := openmeter.NewClient("http://localhost:8888")
	if err != nil {
		panic(err.Error())
	}

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	for {
		// get running pods in all the namespaces with label `subject`
		t := time.Now()
		pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			FieldSelector: "status.phase==Running",
			LabelSelector: "subject",
		})
		if err != nil {
			panic(err.Error())
		}

		for _, pod := range pods.Items {
			// Debug only
			fmt.Printf("time: %s, name: %s, subject: %s\n", t, pod.GetName(), pod.GetLabels()["subject"])

			// Report to OpenMeter
			e := cloudevents.New()
			e.SetID(uuid.New().String())
			e.SetType("pod-runtime")
			e.SetSubject(pod.GetLabels()["subject"])
			e.SetTime(t)
			e.SetSource("kubernetes-api")
			e.SetData("json", map[string]string{
				"name":      pod.GetName(),
				"namespace": pod.GetNamespace(),
			})

			_, err := om.IngestEvents(ctx, e)
			if err != nil {
				panic(err.Error())
			}
		}

		// We report usage every second
		// In OpenMeter we will setup a count aggregation on event type `pod-runtime`, groupped by `name`.
		time.Sleep(1 * time.Second)
	}
}
