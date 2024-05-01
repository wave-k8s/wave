package core

import (
	"context"

	corev1 "k8s.io/api/core/v1"
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
	name := types.NamespacedName{
		Name:      object.GetName(),
		Namespace: object.GetNamespace(),
	}
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

func (h *Handler) watchChildrenForInstance(instance podController, children []configObject) {
	instanceName := types.NamespacedName{
		Name:      instance.GetName(),
		Namespace: instance.GetNamespace(),
	}
	for _, child := range children {
		childName := types.NamespacedName{
			Name:      child.object.GetName(),
			Namespace: child.object.GetNamespace(),
		}

		switch child.object.(type) {
		case *corev1.ConfigMap:
			if _, ok := h.watchedConfigmaps[childName]; !ok {
				h.watchedConfigmaps[childName] = map[types.NamespacedName]bool{}
			}
			h.watchedConfigmaps[childName][instanceName] = true
		case *corev1.Secret:
			if _, ok := h.watchedSecrets[childName]; !ok {
				h.watchedSecrets[childName] = map[types.NamespacedName]bool{}
			}
			h.watchedSecrets[childName][instanceName] = true
		default:
			panic(child.object)
		}
	}
}

func (h *Handler) removeWatchesForInstance(instance podController) {
	instanceName := types.NamespacedName{
		Name:      instance.GetName(),
		Namespace: instance.GetNamespace(),
	}
	h.RemoveWatches(instanceName)
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
