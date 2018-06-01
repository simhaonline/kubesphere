/*
Copyright 2018 The KubeSphere Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package models

import (
	"kubesphere.io/kubesphere/pkg/client"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
	"github.com/golang/glog"
	"strings"
)

const KUBESYSTEM = "kube-system"
const OPENPITRIX = "openpitrix-system"

type Components struct {
	Name         string    `json:"name"`
	Version      string    `json:"version"`
	Kind         string    `json:"kind"`
	Replicas     int       `json:"replicas"`
	HealthStatus string    `json:"healthStatus"`
	SelfLink     string    `json:"selfLink"`
	UpdateTime   time.Time `json:"updateTime"`
}

/***
* get all components from k8s and kubesphere system,
* there are master component, node component,addons component , kubesphere component
*
 */
func GetComponents() (result []Components, err error) {

	k8sClient := client.NewK8sClient()

	label := "tier=control-plane"

	option := meta_v1.ListOptions{

		LabelSelector: label,
	}

	podlists, err := k8sClient.CoreV1().Pods(KUBESYSTEM).List(option)

	if err != nil {

		glog.Error(err)

		return result, err
	}

	var components Components

	templates := [] string{"kube-apiserver", "etcd", "kube-scheduler", "kube-controller-manager", "cloud-controller-manager"}

	if len(podlists.Items) > 0 {

		for _, pod := range podlists.Items {

			for _, template := range templates {

				if strings.Contains(pod.Name, template) {

					components.Name = template
					components.Kind = "Pod"
					components.SelfLink = pod.SelfLink
					version := strings.Split(pod.Spec.Containers[0].Image, ":")

					if len(version) < 2 {

						components.Version = "latest"

					} else {

						components.Version = version[1]

					}
					components.Replicas = 1

					if pod.Status.Phase == "Running" {

						components.HealthStatus = "health"

					} else {

						components.HealthStatus = "unhealth"

					}
					components.UpdateTime = pod.Status.Conditions[0].LastTransitionTime.Time

					result = append(result, components)

				}

			}

		}

	}

	label = "component=kube-addon-manager"

	option.LabelSelector = label

	kubeaddon, err := k8sClient.CoreV1().Pods(KUBESYSTEM).List(option)

	if err != nil {

		glog.Error(err)

		return result, err
	}

	if len(kubeaddon.Items) > 0 {

		for _, pod := range kubeaddon.Items {

			components.Name = "kube-addon-manager"
			components.Kind = "Pod"
			components.SelfLink = pod.SelfLink
			version := strings.Split(pod.Spec.Containers[0].Image, ":")

			if len(version) < 2 {

				components.Version = "latest"

			} else {

				components.Version = version[1]

			}
			components.Replicas = 1

			if pod.Status.Phase == "Running" {

				components.HealthStatus = "health"

			} else {

				components.HealthStatus = "fault"

			}
			components.UpdateTime = pod.Status.Conditions[0].LastTransitionTime.Time

			result = append(result, components)

		}

	}

	option.LabelSelector = ""

	dsList, err := k8sClient.AppsV1beta2().DaemonSets(KUBESYSTEM).List(option)

	if err != nil {

		glog.Error(err)

		return result, err
	}

	if len(dsList.Items) > 0 {

		for _, ds := range dsList.Items {

			if strings.Contains(ds.Name, "fluent-bit") {

				continue
			}

			components.Name = ds.Name
			components.Kind = "Daemonset"
			components.SelfLink = ds.SelfLink
			version := strings.Split(ds.Spec.Template.Spec.Containers[0].Image, ":")

			if len(version) < 2 {

				components.Version = "latest"

			} else {

				components.Version = version[1]

			}
			components.UpdateTime = ds.CreationTimestamp.Time
			components.Replicas = int(ds.Status.DesiredNumberScheduled)

			if ds.Status.NumberAvailable == ds.Status.DesiredNumberScheduled {

				components.HealthStatus = "health"

			} else {

				components.HealthStatus = "fault"

			}

			result = append(result, components)

		}

	}

	templates = []string{"kube-dns", "heapster", "monitoring-influxdb", "iam", "openpitrix"}

	namespaces := []string{KUBESYSTEM, OPENPITRIX}

	for _, ns := range namespaces {

		deployList, err := k8sClient.AppsV1beta1().Deployments(ns).List(option)

		if err != nil {

			glog.Error(err)

			return result, err
		}

		if len(deployList.Items) > 0 {

			for _, dm := range deployList.Items {

				for _, template := range templates {

					if strings.Contains(dm.Name, template) {

						components.Name = dm.Name
						components.Kind = "Deployment"
						components.SelfLink = dm.SelfLink
						components.Replicas = int(dm.Status.Replicas)
						version := strings.Split(dm.Spec.Template.Spec.Containers[0].Image, ":")
						if len(version) < 2 {

							components.Version = "latest"

						} else {

							components.Version = version[1]

						}

						components.UpdateTime = dm.Status.Conditions[0].LastUpdateTime.Time

						if dm.Status.AvailableReplicas == dm.Status.Replicas {

							components.HealthStatus = "health"

						} else {

							components.HealthStatus = "fault"

						}

						result = append(result, components)

					}

				}

			}

		}

	}

	return result, nil

}