package core

import (
	"context"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ handler.EventHandler = &enqueueRequestForWatcher{}

type WatcherList struct {
	watchers      map[types.NamespacedName]map[types.NamespacedName]bool
	watchersMutex *sync.RWMutex
}

type enqueueRequestForWatcher struct {
	WatcherList
}

func EnqueueRequestForWatcher(watcherList WatcherList) handler.EventHandler {
	e := &enqueueRequestForWatcher{
		WatcherList: watcherList,
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
	e.watchersMutex.Lock()
	if watchers, ok := e.watchers[name]; ok {
		for watcher := range watchers {
			request := reconcile.Request{NamespacedName: watcher}
			q.Add(request)
		}
	}
	e.watchersMutex.Unlock()
}

func (h *Handler) GetWatchedConfigmaps() WatcherList {
	return h.watchedConfigmaps
}

func (h *Handler) GetWatchedSecrets() WatcherList {
	return h.watchedSecrets
}

func (h *Handler) watchChildrenForInstance(instance podController, configMaps configMetadataMap, secrets configMetadataMap) {
	instanceName := GetNamespacedNameFromObject(instance)
	h.watchedConfigmaps.watchersMutex.Lock()
	for childName := range configMaps {
		if _, ok := h.watchedConfigmaps.watchers[childName]; !ok {
			h.watchedConfigmaps.watchers[childName] = map[types.NamespacedName]bool{}
		}
		h.watchedConfigmaps.watchers[childName][instanceName] = true
	}
	h.watchedConfigmaps.watchersMutex.Unlock()
	h.watchedSecrets.watchersMutex.Lock()
	for childName := range secrets {
		if _, ok := h.watchedSecrets.watchers[childName]; !ok {
			h.watchedSecrets.watchers[childName] = map[types.NamespacedName]bool{}
		}
		h.watchedSecrets.watchers[childName][instanceName] = true
	}
	h.watchedSecrets.watchersMutex.Unlock()
}

func (h *Handler) removeWatchesForInstance(instance podController) {
	h.RemoveWatches(GetNamespacedNameFromObject(instance))
}

func (h *Handler) RemoveWatches(instanceName types.NamespacedName) {
	h.watchedConfigmaps.watchersMutex.Lock()
	for child, watchers := range h.watchedConfigmaps.watchers {
		delete(watchers, instanceName)
		if len(watchers) == 0 {
			delete(h.watchedConfigmaps.watchers, child)
		}
	}
	h.watchedConfigmaps.watchersMutex.Unlock()
	h.watchedSecrets.watchersMutex.Lock()
	for child, watchers := range h.watchedSecrets.watchers {
		delete(watchers, instanceName)
		if len(watchers) == 0 {
			delete(h.watchedSecrets.watchers, child)
		}
	}
	h.watchedSecrets.watchersMutex.Unlock()
}
