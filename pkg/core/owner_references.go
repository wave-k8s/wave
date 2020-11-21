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
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// removeOwnerReferences iterates over a list of children and removes the owner
// reference from the child before updating it
func (h *Handler) removeOwnerReferences(obj podController, children []Object) error {
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

// updateOwnerReferences determines which children need to have their
// OwnerReferences added/updated and which need to have their OwnerReferences
// removed and then performs all updates
func (h *Handler) updateOwnerReferences(owner podController, existing []Object, current []configObject) error {
	// Add an owner reference to each child object
	errChan := make(chan error)
	for _, obj := range current {
		go func(child Object) {
			errChan <- h.updateOwnerReference(owner, child)
		}(obj.object)
	}

	// Return any errors encountered updating the child objects
	errs := []string{}
	for range current {
		err := <-errChan
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("error(s) encountered updating children: %s", strings.Join(errs, ", "))
	}

	// Get the orphaned children and remove their OwnerReferences
	orphans := getOrphans(existing, current)
	err := h.removeOwnerReferences(owner, orphans)
	if err != nil {
		return fmt.Errorf("error removing Owner References: %v", err)
	}

	return nil
}

// updateOwnerReference ensures that the child object has an OwnerReference
// pointing to the owner
func (h *Handler) updateOwnerReference(owner podController, child Object) error {
	ownerRef := getOwnerReference(owner)
	for _, ref := range child.GetOwnerReferences() {
		// Owner Reference already exists, do nothing
		if reflect.DeepEqual(ref, ownerRef) {
			return nil
		}
	}

	// Append the new OwnerReference and update the child
	h.recorder.Eventf(child, corev1.EventTypeNormal, "AddWatch", "Adding watch for %s %s", kindOf(child), child.GetName())
	ownerRefs := append(child.GetOwnerReferences(), ownerRef)
	child.SetOwnerReferences(ownerRefs)
	err := h.Update(context.TODO(), child)
	if err != nil {
		return fmt.Errorf("error updating child: %v", err)
	}
	return nil
}

// getOrphans creates a slice of orphaned child objects that need their
// OwnerReferences removing
func getOrphans(existing []Object, current []configObject) []Object {
	orphans := []Object{}
	for _, child := range existing {
		if !isIn(current, child) {
			orphans = append(orphans, child)
		}
	}
	return orphans
}

// getOwnerReference constructs an OwnerReference pointing to the object given
func getOwnerReference(obj podController) metav1.OwnerReference {
	t := true
	f := false
	return metav1.OwnerReference{
		APIVersion:         "apps/v1",
		Kind:               kindOf(obj),
		Name:               obj.GetName(),
		UID:                obj.GetUID(),
		BlockOwnerDeletion: &t,
		Controller:         &f,
	}
}

// isIn checks whether a child object exists within a slice of objects
func isIn(list []configObject, child Object) bool {
	for _, obj := range list {
		if obj.object.GetUID() == child.GetUID() {
			return true
		}
	}
	return false
}

// kindOf returns the Kind of the given object as a string
func kindOf(obj Object) string {
	switch obj.(type) {
	case *corev1.ConfigMap:
		return "ConfigMap"
	case *corev1.Secret:
		return "Secret"
	case *deployment:
		return "Deployment"
	case *statefulset:
		return "StatefulSet"
	case *daemonset:
		return "DaemonSet"
	default:
		return "Unknown"
	}
}
