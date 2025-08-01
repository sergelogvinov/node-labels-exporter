/*
Copyright 2025 The Kubernetes Authors.

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

package nodelabelcontroller

import (
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func getEnvsFromNode(node *corev1.Node, pod *corev1.Pod) map[string]string {
	envs := make(map[string]string)

	for k, v := range pod.Annotations {
		if strings.HasPrefix(k, annKeyPrefix) {
			if label, ok := node.Labels[v]; ok {
				env := strings.ToUpper(strings.TrimPrefix(k, annKeyPrefix))
				env = strings.ReplaceAll(env, "-", "_")

				envs[env] = label
			}
		}
	}

	return envs
}

func setEnvsToPod(pod *corev1.Pod, envs map[string]string) {
	for i := range pod.Spec.Containers {
		c := pod.Spec.Containers[i]

		if c.Name != "" {
			for key, value := range envs {
				updated := false

				for j, env := range c.Env {
					if env.Name == key {
						pod.Spec.Containers[i].Env[j].Value = value
						pod.Spec.Containers[i].Env[j].ValueFrom = nil

						updated = true

						break
					}
				}

				if !updated {
					pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, corev1.EnvVar{Name: key, Value: value})
				}
			}
		}
	}
}

func setLabelsToPod(node *corev1.Node, pod *corev1.Pod) map[string]string {
	labels := make(map[string]string)

	for k, v := range pod.Annotations {
		if strings.HasPrefix(k, annKeyPrefix) {
			if label, ok := node.Labels[v]; ok {
				pod.Labels[v] = label
				labels[v] = label
			}
		}
	}

	return labels
}

func setEnvValueFromToPod(pod *corev1.Pod) bool {
	envs := make(map[string]string)
	containers := []string{}

	for k, v := range pod.Annotations {
		if k == annContainers {
			containers = strings.Split(v, ",")
			continue
		}

		if strings.HasPrefix(k, annKeyPrefix) {
			env := strings.ReplaceAll(strings.ToUpper(strings.TrimPrefix(k, annKeyPrefix)), "-", "_")
			envs[env] = v
		}
	}

	if len(envs) == 0 {
		return false
	}

	setEnvValueFromToContainers(pod.Spec.InitContainers, containers, envs)
	setEnvValueFromToContainers(pod.Spec.Containers, containers, envs)

	return true
}

func setEnvValueFromToContainers(items []corev1.Container, containers []string, envs map[string]string) {
	for i := range items {
		c := items[i]

		if len(containers) == 0 || slices.Contains(containers, c.Name) {
			for key, value := range envs {
				updated := false

				envFrom := &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: fmt.Sprintf("metadata.labels['%s']", value),
					},
				}

				for j, env := range c.Env {
					if env.Name == key {
						items[i].Env[j].Value = ""
						items[i].Env[j].ValueFrom = envFrom

						updated = true
					}
				}

				if !updated {
					items[i].Env = append(items[i].Env, corev1.EnvVar{Name: key, ValueFrom: envFrom})
				}
			}
		}
	}
}
