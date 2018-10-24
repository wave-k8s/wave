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
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	opts := client.InNamespace(obj.GetNamespace())

	// List all ConfigMaps in the Deployment's namespace
	configMaps := &corev1.ConfigMapList{}
	err := r.List(context.TODO(), opts, configMaps)
	if err != nil {
		return []metav1.Object{}, fmt.Errorf("error listing ConfigMaps: %v", err)
	}

	// List all Secrets in the Deployment's namespcae
	secrets := &corev1.SecretList{}
	err = r.List(context.TODO(), opts, secrets)
	if err != nil {
		return []metav1.Object{}, fmt.Errorf("error listing Secrets: %v", err)
	}

	// Iterate over the ConfigMaps/Secrets and add the ones owned by the
	// Deployment to the ouput list children
	children := []metav1.Object{}
	for _, cm := range configMaps.Items {
		if isOwnedBy(&cm, obj) {
			children = append(children, cm.DeepCopy())
		}
	}
	for _, s := range secrets.Items {
		if isOwnedBy(&s, obj) {
			children = append(children, s.DeepCopy())
		}
	}

	return children, nil
}

// isOwnedBy returns true if the child has an owner reference that points to
// the owner object
func isOwnedBy(child, owner metav1.Object) bool {
	for _, ref := range child.GetOwnerReferences() {
		if ref.UID == owner.GetUID() {
			return true
		}
	}
	return false
}
