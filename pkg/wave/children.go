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

package wave

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getResult is returned from the getObject method as a helper struct to be
// passed into a channel
type getResult struct {
	err error
	obj Object
}

// getCurrentChildren returns a list of all Secrets and ConfigMaps that are
// referenced in the Deployment's spec
func (h *Handler) getCurrentChildren(obj *appsv1.Deployment) ([]Object, error) {
	configMaps, secrets := getChildNamesByType(obj)

	// get all of ConfigMaps and Secrets
	resultsChan := make(chan getResult)
	for name := range configMaps {
		go func(name string) {
			resultsChan <- h.getConfigMap(obj.GetNamespace(), name)
		}(name)
	}
	for name := range secrets {
		go func(name string) {
			resultsChan <- h.getSecret(obj.GetNamespace(), name)
		}(name)
	}

	// Range over and collect results from the gets
	var errs []string
	var children []Object
	for i := 0; i < len(configMaps)+len(secrets); i++ {
		result := <-resultsChan
		if result.err != nil {
			errs = append(errs, result.err.Error())
		}
		if result.obj != nil {
			children = append(children, result.obj)
		}
	}

	// If there were any errors, don't return any children
	if len(errs) > 0 {
		return []Object{}, fmt.Errorf("error(s) encountered when geting children: %s", strings.Join(errs, ", "))
	}

	// No errors, return the list of children
	return children, nil
}

// getChildNamesByType parses the Depoyment object and returns two sets,
// the first containing the names of all referenced ConfigMaps,
// the second containing the names of all referenced Secrets
func getChildNamesByType(obj *appsv1.Deployment) (map[string]struct{}, map[string]struct{}) {
	// Create sets for storing the names fo the ConfigMaps/Secrets
	configMaps := make(map[string]struct{})
	secrets := make(map[string]struct{})

	// Range through all Volumes and check the VolumeSources for ConfigMaps
	// and Secrets
	for _, vol := range obj.Spec.Template.Spec.Volumes {
		if cm := vol.VolumeSource.ConfigMap; cm != nil {
			configMaps[cm.Name] = struct{}{}
		}
		if s := vol.VolumeSource.Secret; s != nil {
			secrets[s.SecretName] = struct{}{}
		}
	}

	// Range through all Containers and their respective EnvFrom,
	// then check the EnvFromSources for ConfigMaps and Secrets
	for _, container := range obj.Spec.Template.Spec.Containers {
		for _, env := range container.EnvFrom {
			if cm := env.ConfigMapRef; cm != nil {
				configMaps[cm.Name] = struct{}{}
			}
			if s := env.SecretRef; s != nil {
				secrets[s.Name] = struct{}{}
			}
		}
	}

	return configMaps, secrets
}

// getConfigMap gets a ConfigMap with the given name and namespace from the
// API server.
func (h *Handler) getConfigMap(namespace, name string) getResult {
	return h.getObject(namespace, name, &corev1.ConfigMap{})
}

// getSecret gets a Secret with the given name and namespace from the
// API server.
func (h *Handler) getSecret(namespace, name string) getResult {
	return h.getObject(namespace, name, &corev1.Secret{})
}

// getObject gets the Object with the given name and namespace from the API
// server
func (h *Handler) getObject(namespace, name string, obj Object) getResult {
	key := types.NamespacedName{Namespace: namespace, Name: name}
	err := h.Get(context.TODO(), key, obj)
	if err != nil {
		return getResult{err: err}
	}
	return getResult{obj: obj}
}

// getExistingChildren returns a list of all Secrets and ConfigMaps that are
// owned by the Deployment instance
func (h *Handler) getExistingChildren(obj *appsv1.Deployment) ([]Object, error) {
	opts := client.InNamespace(obj.GetNamespace())

	// List all ConfigMaps in the Deployment's namespace
	configMaps := &corev1.ConfigMapList{}
	err := h.List(context.TODO(), opts, configMaps)
	if err != nil {
		return []Object{}, fmt.Errorf("error listing ConfigMaps: %v", err)
	}

	// List all Secrets in the Deployment's namespcae
	secrets := &corev1.SecretList{}
	err = h.List(context.TODO(), opts, secrets)
	if err != nil {
		return []Object{}, fmt.Errorf("error listing Secrets: %v", err)
	}

	// Iterate over the ConfigMaps/Secrets and add the ones owned by the
	// Deployment to the ouput list children
	children := []Object{}
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
