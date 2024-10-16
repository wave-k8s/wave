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
func (e *enqueueRequestForWatcher) Create(ctx context.Context, evt event.CreateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	e.queueOwnerReconcileRequest(evt.Object, q)
}

// Update implements EventHandler.
func (e *enqueueRequestForWatcher) Update(ctx context.Context, evt event.UpdateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	e.queueOwnerReconcileRequest(evt.ObjectOld, q)
	e.queueOwnerReconcileRequest(evt.ObjectNew, q)
}

// Delete implements EventHandler.
func (e *enqueueRequestForWatcher) Delete(ctx context.Context, evt event.DeleteEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	e.queueOwnerReconcileRequest(evt.Object, q)
}

// Generic implements EventHandler.
func (e *enqueueRequestForWatcher) Generic(ctx context.Context, evt event.GenericEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	e.queueOwnerReconcileRequest(evt.Object, q)
}

// queueOwnerReconcileRequest looks the object up in our watchList and queues reconcile.Request to reconcile
// all owners of object
func (e *enqueueRequestForWatcher) queueOwnerReconcileRequest(object metav1.Object, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
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

func (h *Handler[I]) GetWatchedConfigmaps() WatcherList {
	return h.watchedConfigmaps
}

func (h *Handler[I]) GetWatchedSecrets() WatcherList {
	return h.watchedSecrets
}

func (h *Handler[I]) watchChildrenForInstance(instance I, configMaps configMetadataList, secrets configMetadataList) {
	instanceName := GetNamespacedNameFromObject(instance)
	h.watchedConfigmaps.watchersMutex.Lock()
	h.removeWatchedConfigmapsInternal(instanceName)
	for _, child := range configMaps {
		if _, ok := h.watchedConfigmaps.watchers[child.name]; !ok {
			h.watchedConfigmaps.watchers[child.name] = map[types.NamespacedName]bool{}
		}
		h.watchedConfigmaps.watchers[child.name][instanceName] = true
	}
	h.watchedConfigmaps.watchersMutex.Unlock()
	h.watchedSecrets.watchersMutex.Lock()
	h.removeWatchedSecretsInternal(instanceName)
	for _, child := range secrets {
		if _, ok := h.watchedSecrets.watchers[child.name]; !ok {
			h.watchedSecrets.watchers[child.name] = map[types.NamespacedName]bool{}
		}
		h.watchedSecrets.watchers[child.name][instanceName] = true
	}
	h.watchedSecrets.watchersMutex.Unlock()
}

func (h *Handler[I]) removeWatchesForInstance(instance I) {
	h.RemoveWatches(GetNamespacedNameFromObject(instance))
}

func (h *Handler[I]) RemoveWatches(instanceName types.NamespacedName) {
	h.watchedConfigmaps.watchersMutex.Lock()
	h.removeWatchedConfigmapsInternal(instanceName)
	h.watchedConfigmaps.watchersMutex.Unlock()

	h.watchedSecrets.watchersMutex.Lock()
	h.removeWatchedSecretsInternal(instanceName)
	h.watchedSecrets.watchersMutex.Unlock()
}

func (h *Handler[I]) removeWatchedConfigmapsInternal(instanceName types.NamespacedName) {
	for child, watchers := range h.watchedConfigmaps.watchers {
		delete(watchers, instanceName)
		if len(watchers) == 0 {
			delete(h.watchedConfigmaps.watchers, child)
		}
	}
}

func (h *Handler[I]) removeWatchedSecretsInternal(instanceName types.NamespacedName) {
	for child, watchers := range h.watchedSecrets.watchers {
		delete(watchers, instanceName)
		if len(watchers) == 0 {
			delete(h.watchedSecrets.watchers, child)
		}
	}
}
