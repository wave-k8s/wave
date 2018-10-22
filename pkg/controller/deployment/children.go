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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getCurrentChildren returns a list of all Secrets and ConfigMaps that are
// referenced in the Deployment's spec
func (r *ReconcileDeployment) getCurrentChildren(obj *appsv1.Deployment) ([]metav1.Object, error) {
	// TODO: implement this
	return []metav1.Object{}, nil
}

// getExistingChildren returns a list of all Secrets and ConfigMaps that are
// owned by the Deployment instance
func (r *ReconcileDeployment) getExistingChildren(obj *appsv1.Deployment) ([]metav1.Object, error) {
	// TODO: implement this
	return []metav1.Object{}, nil
}
