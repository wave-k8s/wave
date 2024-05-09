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

package daemonset

import (
	"context"

	"github.com/wave-k8s/wave/pkg/core"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=,resources=configmaps,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=,resources=secrets,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=,resources=events,verbs=create;update;patch

// Add creates a new DaemonSet Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	r := newReconciler(mgr)
	return add(mgr, r, r.handler)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) *ReconcileDaemonSet {
	return &ReconcileDaemonSet{
		scheme:  mgr.GetScheme(),
		handler: core.NewHandler[*appsv1.DaemonSet](mgr.GetClient(), mgr.GetEventRecorderFor("wave")),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler, h *core.Handler[*appsv1.DaemonSet]) error {
	return core.AddController("daemonset-controller", &appsv1.DaemonSet{}, mgr, r, h)
}

var _ reconcile.Reconciler = &ReconcileDaemonSet{}

// ReconcileDaemonSet reconciles a DaemonSet object
type ReconcileDaemonSet struct {
	scheme  *runtime.Scheme
	handler *core.Handler[*appsv1.DaemonSet]
}

// Reconcile reads that state of the cluster for a DaemonSet object and
// updates its PodSpec based on mounted configuration
func (r *ReconcileDaemonSet) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	return r.handler.Handle(ctx, request.NamespacedName, &appsv1.DaemonSet{})
}
