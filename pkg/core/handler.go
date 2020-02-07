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

package core

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// Handler performs the main business logic of the Wave controller
type Handler struct {
	client.Client
	recorder record.EventRecorder
	store    inMemoryHashBank
}

// inMemoryHashbank stores the hashes of the children of each podController.
// This is *only* used for debugging: determining what changed, causing wave to
// trigger a rollout.
type inMemoryHashBank struct {
	Bank        map[string]hashEntry //podController[childrenHashes]
	initialised map[string]bool
	sync.RWMutex
}

// hashEntry is what goes in the hashBank. It's a struct as it may make sense to
// change the underlying type later on.
type hashEntry struct {
	entry string
}

// NewHandler constructs a new instance of Handler
func NewHandler(c client.Client, r record.EventRecorder) *Handler {
	hashBank := make(map[string]hashEntry)
	return &Handler{Client: c, recorder: r, store: inMemoryHashBank{Bank: hashBank}}
}

// HandleDeployment is called by the deployment controller to reconcile deployments
func (h *Handler) HandleDeployment(instance *appsv1.Deployment) (reconcile.Result, error) {
	return h.handlePodController(&deployment{Deployment: instance})
}

// HandleStatefulSet is called by the StatefulSet controller to reconcile StatefulSets
func (h *Handler) HandleStatefulSet(instance *appsv1.StatefulSet) (reconcile.Result, error) {
	return h.handlePodController(&statefulset{StatefulSet: instance})
}

// HandleDaemonSet is called by the DaemonSet controller to reconcile DaemonSets
func (h *Handler) HandleDaemonSet(instance *appsv1.DaemonSet) (reconcile.Result, error) {
	return h.handlePodController(&daemonset{DaemonSet: instance})
}

// handlePodController reconciles the state of a podController
func (h *Handler) handlePodController(instance podController) (reconcile.Result, error) {
	log := logf.Log.WithName("wave")

	// If the required annotation isn't present, ignore the instance
	if !hasRequiredAnnotation(instance) {
		// Perform deletion logic if the finalizer is present on the object
		if hasFinalizer(instance) {
			log.V(0).Info("Required annotation removed from instance, cleaning up orphans", "namespace", instance.GetNamespace(), "name", instance.GetName())
			return h.handleDelete(instance)
		}
		return reconcile.Result{}, nil
	}

	// If the instance is marked for deletion, run cleanup process
	if toBeDeleted(instance) {
		log.V(0).Info("Instance marked for deletion, cleaning up orphans", "namespace", instance.GetNamespace(), "name", instance.GetName())
		return h.handleDelete(instance)
	}

	// Get all children that have an OwnerReference pointing to this instance
	existing, err := h.getExistingChildren(instance)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error fetching existing children: %v", err)
	}

	// Get all children that the instance currently references
	current, err := h.getCurrentChildren(instance)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error fetching current children: %v", err)
	}

	// Reconcile the OwnerReferences on the existing and current children
	err = h.updateOwnerReferences(instance, existing, current)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error updating OwnerReferences: %v", err)
	}

	err = h.initialiseHashBank(instance.GetName(), current)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error initialising hashbank for %s: %v", instance.GetName(), err)
	}
	log.V(0).Info("Hashbank Initialised for podController", instance.GetName())

	hash, err := calculateConfigHash(current)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error calculating configuration hash: %v", err)
	}

	// Update the desired state of the Deployment in a DeepCopy
	copy := instance.DeepCopy()
	setConfigHash(copy, hash)
	addFinalizer(copy)

	// If the desired state doesn't match the existing state, update it
	if !reflect.DeepEqual(instance, copy) {
		log.V(0).Info("Updating instance hash", "namespace", instance.GetNamespace(), "name", instance.GetName(), "hash", hash)
		h.recorder.Eventf(copy.GetObject(), corev1.EventTypeNormal, "ConfigChanged", "Configuration hash updated to %s", hash)

		// retrieve the hashes of the children
		oldHashes, err := h.retrieveFromHashBank(current)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("Error retrieving from hashbank: %v", err)
		}

		// calculate the new hashes of the children, update hashbank
		err = h.calculateNewHashBankEntries(instance.GetName(), current)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("Error calculating new hash bank entires for %s: %v", instance.GetName(), err)
		}
		newHashes, err := h.retrieveFromHashBank(current)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("Error retrieving from hashbank: %v", err)
		}

		// diff the two and return the differeing 'name:hash pair(s?)'
		// this only tests hashes for objects that exist in 'oldHashes'
		// I think this is OK?
		for objectName, hash1 := range oldHashes {
			if hash2, ok := newHashes[objectName]; ok {
				if hash2 != hash1 {
					fmt.Printf("Hashes for key %s differ: %s:%s", objectName, hash1, hash2)
					log.V(0).Info("Hashes differ for object", "object", objectName, "hash1", hash1, "hash2", hash2)
					h.recorder.Eventf(copy.GetObject(), corev1.EventTypeNormal, "RolloutTriggered", "A rollout was triggered by %s changing", objectName)
				}
			}
		}

		err = h.Update(context.TODO(), copy.GetObject())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("error updating instance %s/%s: %v", instance.GetNamespace(), instance.GetName(), err)
		}
	}

	return reconcile.Result{}, nil
}
