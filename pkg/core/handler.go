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
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Handler performs the main business logic of the Wave controller
type Handler struct {
	client.Client
	recorder          record.EventRecorder
	watchedConfigmaps WatcherList
	watchedSecrets    WatcherList
}

// NewHandler constructs a new instance of Handler
func NewHandler(c client.Client, r record.EventRecorder) *Handler {
	return &Handler{Client: c, recorder: r,
		watchedConfigmaps: WatcherList{
			watchers:      make(map[types.NamespacedName]map[types.NamespacedName]bool),
			watchersMutex: &sync.RWMutex{},
		},
		watchedSecrets: WatcherList{
			watchers:      make(map[types.NamespacedName]map[types.NamespacedName]bool),
			watchersMutex: &sync.RWMutex{},
		}}
}

// HandleDeployment is called by the deployment controller to reconcile deployments
func (h *Handler) HandleDeployment(instance *appsv1.Deployment) (reconcile.Result, error) {
	return h.handlePodController(&deployment{Deployment: instance})
}

// HandleDeploymentWebhook is called by the deployment webhook
func (h *Handler) HandleDeploymentWebhook(instance *appsv1.Deployment, dryRun *bool, isCreate bool) error {
	return h.updatePodController(&deployment{Deployment: instance}, (dryRun != nil && *dryRun), isCreate)
}

// HandleStatefulSetWebhook is called by the statefulset webhook
func (h *Handler) HandleStatefulSetWebhook(instance *appsv1.StatefulSet, dryRun *bool, isCreate bool) error {
	return h.updatePodController(&statefulset{StatefulSet: instance}, (dryRun != nil && *dryRun), isCreate)
}

// HandleDaemonSetWebhook is called by the daemonset webhook
func (h *Handler) HandleDaemonSetWebhook(instance *appsv1.DaemonSet, dryRun *bool, isCreate bool) error {
	return h.updatePodController(&daemonset{DaemonSet: instance}, (dryRun != nil && *dryRun), isCreate)
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
	configMaps, secrets := getChildNamesByType(instance)
	h.removeWatchesForInstance(instance)
	h.watchChildrenForInstance(instance, configMaps, secrets)

	// Get content of children
	current, err := h.getCurrentChildren(configMaps, secrets)
	if err != nil {
		if _, ok := err.(*NotFoundError); ok {
			// We are missing children but we added watchers for all children so we are done
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("error fetching current children: %v", err)
	}

	hash, err := calculateConfigHash(current)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error calculating configuration hash: %v", err)
	}

	// Update the desired state of the Deployment in a DeepCopy
	copy := instance.DeepCopyPodController()
	setConfigHash(copy, hash)

	if isSchedulingDisabled(copy) {
		restoreScheduling(copy)
	}

	// If the desired state doesn't match the existing state, update it
	if !reflect.DeepEqual(instance, copy) {
		log.V(0).Info("Updating instance hash", "hash", hash)
		h.recorder.Eventf(copy.GetApiObject(), corev1.EventTypeNormal, "ConfigChanged", "Configuration hash updated to %s", hash)

		err := h.Update(context.TODO(), copy.GetApiObject())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("error updating instance %s/%s: %v", instance.GetNamespace(), instance.GetName(), err)
		}
	}
	return reconcile.Result{}, nil
}

// handlePodController will only update the hash. Everything else is left to the reconciler.
func (h *Handler) updatePodController(instance podController, dryRun bool, isCreate bool) error {
	log := logf.Log.WithName("wave").WithValues("namespace", instance.GetNamespace(), "name", instance.GetName(), "dryRun", dryRun, "isCreate", isCreate)
	log.V(5).Info("Running webhook")

	// If the required annotation isn't present, ignore the instance
	if !hasRequiredAnnotation(instance) {
		return nil
	}

	// Get all children that the instance currently references
	configMaps, secrets := getChildNamesByType(instance)
	current, err := h.getCurrentChildren(configMaps, secrets)
	if err != nil {
		if _, ok := err.(*NotFoundError); ok {
			if isCreate {
				log.V(0).Info("Not all required children found yet. Disabling scheduling!", "err", err)
				disableScheduling(instance)
			} else {
				log.V(0).Info("Not all required children found yet. Skipping mutation!", "err", err)
			}
			return nil
		} else {
			return fmt.Errorf("error fetching current children: %v", err)
		}
	}

	hash, err := calculateConfigHash(current)
	if err != nil {
		return fmt.Errorf("error calculating configuration hash: %v", err)
	}

	// Update the desired state of the Deployment
	oldHash := getConfigHash(instance)
	setConfigHash(instance, hash)

	if !dryRun && oldHash != hash {
		log.V(0).Info("Updating instance hash", "hash", hash)
		h.recorder.Eventf(instance.GetApiObject(), corev1.EventTypeNormal, "ConfigChanged", "Configuration hash updated to %s", hash)
	}

	return nil
}
