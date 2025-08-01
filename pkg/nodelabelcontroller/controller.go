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
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// NodeLabelsEnvInjector injects node labels to pod environment variables
type NodeLabelsEnvInjector struct {
	client  kubernetes.Interface
	log     logr.Logger
	decoder admission.Decoder

	nodeLister corelisters.NodeLister
}

// NewNodeLabelsEnvInjector creates a new NodeLabelsEnvInjector
func NewNodeLabelsEnvInjector(client kubernetes.Interface, scheme *runtime.Scheme, nodeLister corelisters.NodeLister, log logr.Logger) *NodeLabelsEnvInjector {
	return &NodeLabelsEnvInjector{
		client:     client,
		log:        log,
		decoder:    admission.NewDecoder(scheme),
		nodeLister: nodeLister,
	}
}

// Handle handles the admission request
func (i *NodeLabelsEnvInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	if req.Operation != admissionv1.Create {
		return admission.Allowed("not a Create request")
	}

	i.log.V(1).Info("Handling request", "kind", req.RequestKind.Kind, "namespace", req.Namespace, "uid", req.UID)

	if req.RequestKind.Kind == "Pod" {
		pod := &corev1.Pod{}
		if err := i.decoder.Decode(req, pod); err != nil {
			i.log.Error(err, "Failed to decode request object")

			return admission.Errored(http.StatusBadRequest, err)
		}

		name := pod.Name
		if name == "" {
			name = pod.GenerateName
		}

		i.log.V(1).Info("Handling request", "namespace", pod.Namespace, "name", name)

		if !setEnvValueFromToPod(pod) {
			return admission.Allowed("skipped")
		}

		podRaw, err := json.Marshal(pod)
		if err != nil {
			i.log.Error(err, "Failed to encode pod object")

			return admission.Errored(http.StatusInternalServerError, err)
		}

		i.log.Info("Injecting envFrom to pod", "namespace", pod.Namespace, "name", name)

		return admission.PatchResponseFromRaw(req.Object.Raw, podRaw)
	}

	if req.RequestKind.Kind == "Binding" {
		binding := &corev1.Binding{}
		if err := json.Unmarshal(req.Object.Raw, binding); err != nil {
			i.log.Error(err, "Failed to decode request object")

			return admission.Errored(http.StatusBadRequest, fmt.Errorf("json unmarshal Binding with error: %v", err))
		}

		if binding.Target.Kind != "Node" || binding.Target.Name == "" {
			i.log.Info("Pod binding target is not Node or target name empty", "binding", binding)

			return admission.Allowed("skipped")
		}

		i.log.V(1).Info("Handling request", "node", binding.Target.Name, "namespace", binding.Namespace, "name", binding.Name)

		pod, err := i.client.CoreV1().Pods(binding.Namespace).Get(ctx, binding.Name, metav1.GetOptions{})
		if err != nil {
			i.log.Error(err, "Failed to get pod", "namespace", binding.Namespace, "name", binding.Name)

			return admission.Errored(http.StatusInternalServerError, fmt.Errorf("failed to get pod %s/%s: %v", binding.Namespace, binding.Name, err))
		}

		node, err := i.nodeLister.Get(binding.Target.Name)
		if err != nil {
			i.log.Error(err, "Failed to get node", "node", binding.Target.Name)

			return admission.Errored(http.StatusInternalServerError, fmt.Errorf("failed to get node %s: %v", binding.Target.Name, err))
		}

		updated := pod.DeepCopy()

		labels := setLabelsToPod(node, updated)
		if len(labels) == 0 {
			return admission.Allowed("skipped")
		}

		i.log.Info("Injecting node labels to pod", "namespace", binding.Namespace, "name", binding.Name, "labels", labels)

		updatedBytes, err := json.Marshal(updated)
		if err != nil {
			i.log.Error(err, "Failed to encode new pod object")

			return admission.Errored(http.StatusInternalServerError, err)
		}

		podBytes, err := json.Marshal(pod)
		if err != nil {
			i.log.Error(err, "Failed to encode old pod object")

			return admission.Errored(http.StatusInternalServerError, err)
		}

		patchBytes, err := strategicpatch.CreateTwoWayMergePatch(podBytes, updatedBytes, &corev1.Pod{})
		if err != nil {
			i.log.Error(err, "Failed to create patch")

			return admission.Errored(http.StatusInternalServerError, err)
		}

		_, err = i.client.CoreV1().Pods(binding.Namespace).Patch(ctx, binding.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
		if err != nil {
			i.log.Error(err, "Failed to patch pod", "namespace", binding.Namespace, "name", binding.Name)

			return admission.Errored(http.StatusInternalServerError, err)
		}

		return admission.Allowed("patched")
	}

	return admission.Allowed("done")
}
