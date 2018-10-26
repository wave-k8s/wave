/*
Copyright 2018 Pusher Ltd.

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

package deployment

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// removeOwnerReferences iterates over a list of children and removes the owner
// reference from the child before updating it
func (r *ReconcileDeployment) removeOwnerReferences(obj *appsv1.Deployment, children []object) error {
	for _, child := range children {
		// Filter the existing ownerReferences
		ownerRefs := []metav1.OwnerReference{}
		for _, ref := range child.GetOwnerReferences() {
			if ref.UID != obj.UID {
				ownerRefs = append(ownerRefs, ref)
			}
		}

		// Compare the ownerRefs and update if they have changed
		if !reflect.DeepEqual(ownerRefs, child.GetOwnerReferences()) {
			child.SetOwnerReferences(ownerRefs)
			err := r.Update(context.TODO(), child)
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
func (r *ReconcileDeployment) updateOwnerReferences(owner *appsv1.Deployment, existing, current []object) error {
	// Add an owner reference to each child object
	errChan := make(chan error)
	for _, obj := range current {
		go func(child object) {
			errChan <- r.updateOwnerReference(owner, child)
		}(obj)
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
	err := r.removeOwnerReferences(owner, orphans)
	if err != nil {
		return fmt.Errorf("error removing Owner References: %v", err)
	}

	return nil
}

// updateOwnerReference ensures that the child object has an OwnerReference
// pointing to the owner
func (r *ReconcileDeployment) updateOwnerReference(owner, child object) error {
	return fmt.Errorf("NOT YET IMPLEMENTED")
}

// getOrphans creates a slice of orphaned child objects that need their
// OwnerReferences removing
func getOrphans(existing, current []object) []object {
	return []object{}
}
