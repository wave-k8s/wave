/*
Copyright 2018 Pusher Ltd. and Wave Contributors

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

package core

import (
	"context"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// removeOwnerReferences iterates over a list of children and removes the owner
// reference from the child before updating it
func (h *Handler[I]) removeOwnerReferences(obj I, children []Object) error {
	for _, child := range children {
		// Filter the existing ownerReferences
		ownerRefs := []metav1.OwnerReference{}
		for _, ref := range child.GetOwnerReferences() {
			if ref.UID != obj.GetUID() {
				ownerRefs = append(ownerRefs, ref)
			}
		}

		// Compare the ownerRefs and update if they have changed
		if !reflect.DeepEqual(ownerRefs, child.GetOwnerReferences()) {
			h.recorder.Eventf(child, corev1.EventTypeNormal, "RemoveWatch", "Removing watch for %s %s", kindOf(child), child.GetName())
			child.SetOwnerReferences(ownerRefs)
			err := h.Update(context.TODO(), child)
			if err != nil {
				return fmt.Errorf("error updating child %s/%s: %v", child.GetNamespace(), child.GetName(), err)
			}
		}
	}
	return nil
}

// kindOf returns the Kind of the given object as a string
func kindOf(obj Object) string {
	switch obj.(type) {
	case *corev1.ConfigMap:
		return "ConfigMap"
	case *corev1.Secret:
		return "Secret"
	case *appsv1.Deployment:
		return "Deployment"
	case *appsv1.StatefulSet:
		return "StatefulSet"
	case *appsv1.DaemonSet:
		return "DaemonSet"
	default:
		return "Unknown"
	}
}
