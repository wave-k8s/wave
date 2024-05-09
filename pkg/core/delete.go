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

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// deleteOwnerReferencesAndFinalizer removes all existing Owner References pointing to the object
// before removing the object's Finalizer
func (h *Handler[I]) deleteOwnerReferencesAndFinalizer(obj I) (reconcile.Result, error) {
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
	if hasFinalizer(obj) {
		removeFinalizer(obj)
		err := h.Update(context.TODO(), obj)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("error updating Deployment: %v", err)
		}
	}
	return reconcile.Result{}, nil
}
