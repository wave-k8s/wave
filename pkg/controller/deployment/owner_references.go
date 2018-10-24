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

package deployment

import (
	appsv1 "k8s.io/api/apps/v1"
)

// removeOwnerReferences iterates over a list of children and removes the owner
// reference from the child before updating it
func (r *ReconcileDeployment) removeOwnerReferences(obj *appsv1.Deployment, children []object) error {
	return nil
}

// updateOwnerReferences determines which children need to have their
// OwnerReferences added/updated and which need to have their OwnerReferences
// removed and then performs all updates
func (r *ReconcileDeployment) updateOwnerReferences(obj *appsv1.Deployment, existing, current []object) error {
	// TODO: implement this
	return nil
}
