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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// handleDelete removes all existing Owner References pointing to the object
// before removing the object's Finalizer
func (h *Handler) handleDelete(obj podController) (reconcile.Result, error) {
	// Fetch all children with an OwnerReference pointing to the object
	existing, err := h.getExistingChildren(obj)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error fetching children: %v", err)
	}

	// Remove the OwnerReferences from the children
	err = h.removeOwnerReferences(obj, existing)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error removing owner references from children: %v", err)
	}

	// Remove the object's Finalizer and update if necessary
	copy := obj.DeepCopyPodController()
	removeFinalizer(copy)
	if !reflect.DeepEqual(obj, copy) {
		err := h.Update(context.TODO(), copy.GetApiObject())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("error updating Deployment: %v", err)
		}
	}
	return reconcile.Result{}, nil
}

// toBeDeleted checks whether the object has been marked for deletion
func toBeDeleted(obj metav1.Object) bool {
	// IsZero means that the object hasn't been marked for deletion
	return !obj.GetDeletionTimestamp().IsZero()
}
