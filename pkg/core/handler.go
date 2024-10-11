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
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Handler performs the main business logic of the Wave controller
type Handler[I InstanceType] struct {
	client.Client
	recorder          record.EventRecorder
	watchedConfigmaps WatcherList
	watchedSecrets    WatcherList
}

// NewHandler constructs a new instance of Handler
func NewHandler[I InstanceType](c client.Client, r record.EventRecorder) *Handler[I] {
	return &Handler[I]{Client: c, recorder: r,
		watchedConfigmaps: WatcherList{
			watchers:      make(map[types.NamespacedName]map[types.NamespacedName]bool),
			watchersMutex: &sync.RWMutex{},
		},
		watchedSecrets: WatcherList{
			watchers:      make(map[types.NamespacedName]map[types.NamespacedName]bool),
			watchersMutex: &sync.RWMutex{},
		}}
}

// HandleWebhook is called by the webhook
func (h *Handler[I]) HandleWebhook(instance I, dryRun *bool, isCreate bool) error {
	return h.updatePodController(instance, (dryRun != nil && *dryRun), isCreate)
}

// Handle is called by the controller to reconcile its object
func (h *Handler[I]) Handle(ctx context.Context, namespacesName types.NamespacedName, instance I) (reconcile.Result, error) {
	err := h.Get(ctx, namespacesName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			h.RemoveWatches(namespacesName)
			// Object not found, return.  Created objects are automatically garbage collected.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	return h.handlePodController(instance)
}

// handlePodController reconciles the state of a podController
func (h *Handler[I]) handlePodController(instance I) (reconcile.Result, error) {
	log := logf.Log.WithName("wave").WithValues("namespace", instance.GetNamespace(), "name", instance.GetName())

	// To cleanup legacy ownerReferences and finalizer
	if hasFinalizer(instance) {
		log.V(0).Info("Removing old finalizer")
		return h.deleteOwnerReferencesAndFinalizer(instance)
	}

	// If the required annotation isn't present, ignore the instance
	if !hasRequiredAnnotation(instance) {
		h.removeWatchesForInstance(instance)
		return reconcile.Result{}, nil
	}

	log.V(5).Info("Reconciling")

	// Get all children and add watches
	configMapsConfig, secretsConfig := getChildNamesByType(instance)
	h.watchChildrenForInstance(instance, configMapsConfig, secretsConfig)

	// Get content of children
	configMaps, secrets, err := h.getCurrentChildren(configMapsConfig, secretsConfig)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error fetching current children: %v", err)
	}

	err = h.checkRequiredChildren(configMaps, secrets, configMapsConfig, secretsConfig)
	if err != nil {
		// We are missing children but we added watchers for all children so we are done
		log.V(0).Info("Waiting for children...", "error", err)
		return reconcile.Result{}, nil
	}

	log.V(0).Info("All children found", "configMaps", fmt.Sprint(configMaps), "secrets", fmt.Sprint(secrets))

	hash, err := calculateConfigHash(configMaps, secrets, configMapsConfig, secretsConfig)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error calculating configuration hash: %v", err)
	}

	// Update the desired state of the Deployment in a DeepCopy
	oldHash := getConfigHash(instance)
	setConfigHash(instance, hash)

	schedulingChange := false
	if isSchedulingDisabled(instance) {
		log.V(0).Info("Enabled scheduling since all children became available.")
		h.recorder.Eventf(instance, corev1.EventTypeNormal, "SchedulingEnabled", "Enabled scheduling since all children became available.")
		restoreScheduling(instance)
		schedulingChange = true
	}

	// If the desired state doesn't match the existing state, update it
	if hash != oldHash || schedulingChange {
		log.V(0).Info("Updating instance hash", "hash", hash)
		h.recorder.Eventf(instance, corev1.EventTypeNormal, "ConfigChanged", "Configuration hash updated to %s", hash)

		err := h.Update(context.TODO(), instance)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("error updating instance %s/%s: %v", instance.GetNamespace(), instance.GetName(), err)
		}
	}
	return reconcile.Result{}, nil
}

// handlePodController will only update the hash. Everything else is left to the reconciler.
func (h *Handler[I]) updatePodController(instance I, dryRun bool, isCreate bool) error {
	log := logf.Log.WithName("wave").WithValues("namespace", instance.GetNamespace(), "name", instance.GetName(), "dryRun", dryRun, "isCreate", isCreate)
	log.V(5).Info("Running webhook")

	// If the required annotation isn't present, ignore the instance
	if !hasRequiredAnnotation(instance) {
		return nil
	}

	// Get all children that the instance currently references
	configMapsConfig, secretsConfig := getChildNamesByType(instance)
	configMaps, secrets, err := h.getCurrentChildren(configMapsConfig, secretsConfig)
	if err != nil {
		return fmt.Errorf("error fetching current children: %v", err)
	}

	err = h.checkRequiredChildren(configMaps, secrets, configMapsConfig, secretsConfig)
	if err != nil {
		if isCreate {
			if !dryRun {
				log.V(0).Info("Not all required children found yet. Disabling scheduling!", "err", err)
				h.recorder.Eventf(instance, corev1.EventTypeNormal, "SchedulingDisabled", "Disabled scheduling due to missing children: %s", err)
			}
			disableScheduling(instance)
		} else {
			log.V(0).Info("Not all required children found yet. Skipping mutation!", "err", err)
		}
		return nil
	}

	hash, err := calculateConfigHash(configMaps, secrets, configMapsConfig, secretsConfig)
	if err != nil {
		return fmt.Errorf("error calculating configuration hash: %v", err)
	}

	// Update the desired state of the Deployment
	oldHash := getConfigHash(instance)
	setConfigHash(instance, hash)

	if !dryRun && oldHash != hash {
		log.V(0).Info("Updating instance hash", "hash", hash)
		h.recorder.Eventf(instance, corev1.EventTypeNormal, "ConfigChanged", "Configuration hash updated to %s", hash)
	}

	return nil
}
