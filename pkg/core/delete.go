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
	copy := obj.DeepCopy()
	removeFinalizer(copy)
	if !reflect.DeepEqual(obj, copy) {
		err := h.Update(context.TODO(), copy.GetObject())
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
