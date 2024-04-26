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

package statefulset

import (
	"context"

	"github.com/wave-k8s/wave/pkg/core"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new StatefulSet Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileStatefulSet{
		scheme:  mgr.GetScheme(),
		handler: core.NewHandler(mgr.GetClient(), mgr.GetEventRecorderFor("wave")),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("statefulset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to StatefulSet
	err = c.Watch(source.Kind(mgr.GetCache(), &appsv1.StatefulSet{}), &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	handler := handler.EnqueueRequestForOwner(mgr.GetScheme(), mgr.GetRESTMapper(), &appsv1.StatefulSet{})

	// Watch ConfigMaps owned by a StatefulSet
	err = c.Watch(source.Kind(mgr.GetCache(), &corev1.ConfigMap{}), handler)
	if err != nil {
		return err
	}

	// Watch Secrets owned by a StatefulSet
	err = c.Watch(source.Kind(mgr.GetCache(), &corev1.Secret{}), handler)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileStatefulSet{}

// ReconcileStatefulSet reconciles a StatefulSet object
type ReconcileStatefulSet struct {
	scheme  *runtime.Scheme
	handler *core.Handler
}

// Reconcile reads that state of the cluster for a StatefulSet object and
// updates its PodSpec based on mounted configuration
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=,resources=configmaps,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=,resources=secrets,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=,resources=events,verbs=create;update;patch
func (r *ReconcileStatefulSet) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// Fetch the StatefulSet instance
	instance := &appsv1.StatefulSet{}
	err := r.handler.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	return r.handler.HandleStatefulSet(instance)
}
