package core

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ handler.EventHandler = &enqueueRequestForWatcher{}

type enqueueRequestForWatcher struct {
	// watcherList
	watcherList map[types.NamespacedName]map[types.NamespacedName]bool
}

func EnqueueRequestForWatcher(watcherList map[types.NamespacedName]map[types.NamespacedName]bool) handler.EventHandler {
	e := &enqueueRequestForWatcher{
		watcherList: watcherList,
	}
	return e
}

// Create implements EventHandler.
func (e *enqueueRequestForWatcher) Create(ctx context.Context, evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	e.queueOwnerReconcileRequest(evt.Object, q)
}

// Update implements EventHandler.
func (e *enqueueRequestForWatcher) Update(ctx context.Context, evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	e.queueOwnerReconcileRequest(evt.ObjectOld, q)
	e.queueOwnerReconcileRequest(evt.ObjectNew, q)
}

// Delete implements EventHandler.
func (e *enqueueRequestForWatcher) Delete(ctx context.Context, evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	e.queueOwnerReconcileRequest(evt.Object, q)
}

// Generic implements EventHandler.
func (e *enqueueRequestForWatcher) Generic(ctx context.Context, evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	e.queueOwnerReconcileRequest(evt.Object, q)
}

// queueOwnerReconcileRequest looks the object up in our watchList and queues reconcile.Request to reconcile
// all owners of object
func (e *enqueueRequestForWatcher) queueOwnerReconcileRequest(object metav1.Object, q workqueue.RateLimitingInterface) {
	name := GetNamespacedNameFromObject(object)
	if watchers, ok := e.watcherList[name]; ok {
		for watcher := range watchers {
			request := reconcile.Request{NamespacedName: watcher}
			q.Add(request)
		}
	}
}

func (h *Handler) GetWatchedConfigmaps() map[types.NamespacedName]map[types.NamespacedName]bool {
	return h.watchedConfigmaps
}

func (h *Handler) GetWatchedSecrets() map[types.NamespacedName]map[types.NamespacedName]bool {
	return h.watchedSecrets
}

func (h *Handler) watchChildrenForInstance(instance podController, configMaps configMetadataMap, secrets configMetadataMap) {
	instanceName := GetNamespacedNameFromObject(instance)
	for childName := range configMaps {

		if _, ok := h.watchedConfigmaps[childName]; !ok {
			h.watchedConfigmaps[childName] = map[types.NamespacedName]bool{}
		}
		h.watchedConfigmaps[childName][instanceName] = true
	}
	for childName := range secrets {
		if _, ok := h.watchedSecrets[childName]; !ok {
			h.watchedSecrets[childName] = map[types.NamespacedName]bool{}
		}
		h.watchedSecrets[childName][instanceName] = true
	}
}

func (h *Handler) removeWatchesForInstance(instance podController) {
	h.RemoveWatches(GetNamespacedNameFromObject(instance))
}

func (h *Handler) RemoveWatches(instanceName types.NamespacedName) {
	for child, watchers := range h.watchedConfigmaps {
		delete(watchers, instanceName)
		if len(watchers) == 0 {
			delete(h.watchedConfigmaps, child)
		}
	}
	for child, watchers := range h.watchedSecrets {
		delete(watchers, instanceName)
		if len(watchers) == 0 {
			delete(h.watchedSecrets, child)
		}
	}
}
