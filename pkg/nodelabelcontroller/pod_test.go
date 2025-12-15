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
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_annotationKeyToEnvName(t *testing.T) {
	for _, tt := range []struct {
		name     string
		key      string
		expected string
		ok       bool
	}{
		{
			name:     "valid annotation key",
			key:      annKeyPrefix + "region",
			expected: "REGION",
			ok:       true,
		},
		{
			name:     "valid annotation key with hyphen",
			key:      annKeyPrefix + "node-zone",
			expected: "NODE_ZONE",
			ok:       true,
		},
		{
			name:     "invalid annotation key",
			key:      "some.other.key",
			expected: "",
			ok:       false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			env, ok := annotationKeyToEnvName(tt.key)
			assert.Equal(t, tt.expected, env)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

func Test_getEnvsFromNode(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node0",
			Labels: map[string]string{
				"topology.kubernetes.io/region": "region-1",
				"topology.kubernetes.io/zone":   "zone-1",
				"custom-label":                  "custom-value",
			},
		},
	}

	for _, tt := range []struct {
		name     string
		node     *corev1.Node
		pod      *corev1.Pod
		expected map[string]string
	}{
		{
			name: "pod without annotations",
			node: node,
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod0",
				},
			},
			expected: map[string]string{},
		},
		{
			name: "pod with region and zone",
			node: node,
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod0",
					Annotations: map[string]string{
						annKeyPrefix + "region":    "topology.kubernetes.io/region",
						annKeyPrefix + "node-zone": "topology.kubernetes.io/zone",
					},
				},
			},
			expected: map[string]string{
				"REGION":    "region-1",
				"NODE_ZONE": "zone-1",
			},
		},
		{
			name: "pod with region and non-existing label",
			node: node,
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod0",
					Annotations: map[string]string{
						annKeyPrefix + "region": "topology.kubernetes.io/region",
						annKeyPrefix + "test":   "test-label",
					},
				},
			},
			expected: map[string]string{
				"REGION": "region-1",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, getEnvsFromNode(tt.node, tt.pod))
		})
	}
}

func Test_setEnvsToPod(t *testing.T) {
	podEnvSecret := corev1.EnvVar{
		Name: "ENV2",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "secret",
				},
				Key: "key",
			},
		},
	}

	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container0",
					Env: []corev1.EnvVar{
						{
							Name:  "ENV1",
							Value: "value1",
						},
						podEnvSecret,
					},
				},
			},
		},
	}

	podTwo := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container0",
					Env: []corev1.EnvVar{
						{
							Name:  "ENV1",
							Value: "value1",
						},
						podEnvSecret,
					},
				},
				{
					Name: "container1",
				},
			},
		},
	}

	for _, tt := range []struct {
		name     string
		pod      *corev1.Pod
		envs     map[string]string
		expected *corev1.Pod
	}{
		{
			name: "pod without envs",
			pod:  pod,
			envs: map[string]string{},
			expected: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container0",
							Env: []corev1.EnvVar{
								{
									Name:  "ENV1",
									Value: "value1",
								},
								podEnvSecret,
							},
						},
					},
				},
			},
		},
		{
			name: "pod with envs",
			pod:  pod,
			envs: map[string]string{
				"ENV3": "value3",
			},
			expected: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container0",
							Env: []corev1.EnvVar{
								{
									Name:  "ENV1",
									Value: "value1",
								},
								podEnvSecret,
								{
									Name:  "ENV3",
									Value: "value3",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "pod with envs override",
			pod:  pod,
			envs: map[string]string{
				"ENV2": "value3",
			},
			expected: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container0",
							Env: []corev1.EnvVar{
								{
									Name:  "ENV1",
									Value: "value1",
								},
								{
									Name:  "ENV2",
									Value: "value3",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "pod with two containers",
			pod:  podTwo,
			envs: map[string]string{
				"ENV3": "value3",
			},
			expected: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container0",
							Env: []corev1.EnvVar{
								{
									Name:  "ENV1",
									Value: "value1",
								},
								podEnvSecret,
								{
									Name:  "ENV3",
									Value: "value3",
								},
							},
						},
						{
							Name: "container1",
							Env: []corev1.EnvVar{
								{
									Name:  "ENV3",
									Value: "value3",
								},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			newPod := tt.pod.DeepCopy()
			setEnvsToPod(newPod, tt.envs)
			assert.Equal(t, tt.expected, newPod)
		})
	}
}

func Test_setEnvValueFromToPod(t *testing.T) {
	podEnvSecret := corev1.EnvVar{
		Name: "ENV2",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "secret",
				},
				Key: "key",
			},
		},
	}

	for _, tt := range []struct {
		name     string
		pod      *corev1.Pod
		expected *corev1.Pod
	}{
		{
			name: "pod without envs",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod0",
					Annotations: map[string]string{
						annKeyPrefix + "zone":     "value1",
						annKeyPrefix + "test-env": "value2",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container0",
						},
					},
				},
			},
			expected: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod0",
					Annotations: map[string]string{
						annKeyPrefix + "zone":     "value1",
						annKeyPrefix + "test-env": "value2",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container0",
							Env: []corev1.EnvVar{
								{
									Name: "ZONE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.labels['value1']",
										},
									},
								},
								{
									Name: "TEST_ENV",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.labels['value2']",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "pod with envs",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod0",
					Annotations: map[string]string{
						annKeyPrefix + "zone": "value1",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container0",
							Env: []corev1.EnvVar{
								{
									Name:  "ENV1",
									Value: "value1",
								},
								podEnvSecret,
							},
						},
					},
				},
			},
			expected: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod0",
					Annotations: map[string]string{
						annKeyPrefix + "zone": "value1",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container0",
							Env: []corev1.EnvVar{
								{
									Name:  "ENV1",
									Value: "value1",
								},
								podEnvSecret,
								{
									Name: "ZONE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.labels['value1']",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "pod with envs and specified container",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod0",
					Annotations: map[string]string{
						annKeyPrefix + "zone": "value1",
						annContainers:         "container0",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container0",
							Env: []corev1.EnvVar{
								{
									Name:  "ENV1",
									Value: "value1",
								},
								podEnvSecret,
							},
						},
						{
							Name: "container1",
							Env: []corev1.EnvVar{
								{
									Name:  "ENV1",
									Value: "value1",
								},
								podEnvSecret,
							},
						},
					},
				},
			},
			expected: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod0",
					Annotations: map[string]string{
						annKeyPrefix + "zone": "value1",
						annContainers:         "container0",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container0",
							Env: []corev1.EnvVar{
								{
									Name:  "ENV1",
									Value: "value1",
								},
								podEnvSecret,
								{
									Name: "ZONE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.labels['value1']",
										},
									},
								},
							},
						},
						{
							Name: "container1",
							Env: []corev1.EnvVar{
								{
									Name:  "ENV1",
									Value: "value1",
								},
								podEnvSecret,
							},
						},
					},
				},
			},
		},
		{
			name: "pod with envs and init container",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod0",
					Annotations: map[string]string{
						annKeyPrefix + "zone": "value1",
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name: "init-container1",
							Env: []corev1.EnvVar{
								{
									Name:  "ENV1",
									Value: "value1",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "container0",
							Env: []corev1.EnvVar{
								{
									Name:  "ENV1",
									Value: "value1",
								},
								podEnvSecret,
							},
						},
					},
				},
			},
			expected: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod0",
					Annotations: map[string]string{
						annKeyPrefix + "zone": "value1",
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name: "init-container1",
							Env: []corev1.EnvVar{
								{
									Name:  "ENV1",
									Value: "value1",
								},
								{
									Name: "ZONE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.labels['value1']",
										},
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name: "container0",
							Env: []corev1.EnvVar{
								{
									Name:  "ENV1",
									Value: "value1",
								},
								podEnvSecret,
								{
									Name: "ZONE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.labels['value1']",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			newPod := tt.pod.DeepCopy()
			setEnvValueFromToPod(newPod)
			assert.Equal(t, tt.expected, newPod)
		})
	}
}
