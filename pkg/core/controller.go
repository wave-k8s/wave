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
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func AddController[I InstanceType](name string, typeInstance I, mgr manager.Manager, r reconcile.Reconciler, h *Handler[I]) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(typeInstance).
		Watches(&corev1.ConfigMap{}, EnqueueRequestForWatcher(h.GetWatchedConfigmaps())).
		Watches(&corev1.Secret{}, EnqueueRequestForWatcher(h.GetWatchedSecrets())).
		Complete(r)
}
