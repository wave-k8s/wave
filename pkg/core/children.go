/*
Copyright 2018, 2019 Pusher Ltd.

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
	err           error
	obj           Object
	singleField   bool
	fieldKey      string
	fieldOptional bool
}

// getCurrentChildren returns a list of all Secrets and ConfigMaps that are
// referenced in the Deployment's spec.  Any reference to a whole ConfigMap or Secret
// (i.e. via an EnvFrom or a Volume) will result in one entry in the list, irrespective of
// whether individual elements are also references (i.e. via an Env entry).
func (h *Handler) getCurrentChildren(obj *appsv1.Deployment) ([]ConfigObject, error) {
	configMaps, secrets, configMapKeyReferences, secretKeyReferences := getChildNamesByType(obj)
	var childCount int

	// get all of ConfigMaps and Secrets
	resultsChan := make(chan getResult)
	for name := range configMaps {
		childCount++
		go func(name string) {
			resultsChan <- h.getConfigMap(obj.GetNamespace(), name)
		}(name)
	}
	for name := range secrets {
		childCount++
		go func(name string) {
			resultsChan <- h.getSecret(obj.GetNamespace(), name)
		}(name)
	}
	for name := range configMapKeyReferences {
		// Only include ConfigMaps that are not already pulled in as a whole via EnvFrom/Volumes
		if _, ok := configMaps[name]; !ok {
			for key, config := range configMapKeyReferences[name] {
				childCount++
				go func(name string, key string, optional bool) {
					resultsChan <- h.getConfigMapWithKey(obj.GetNamespace(), name, key, optional)
				}(name, key, config.optional)
			}
		}
	}
	for name := range secretKeyReferences {
		// Only include Secrets that are not already pulled in as a whole via EnvFrom/Volumes
		if _, ok := secrets[name]; !ok {
			for key, config := range secretKeyReferences[name] {
				childCount++
				go func(name string, key string, optional bool) {
					resultsChan <- h.getSecretWithKey(obj.GetNamespace(), name, key, optional)
				}(name, key, config.optional)
			}
		}
	}

	// Range over and collect results from the gets
	var errs []string
	var childMap = make(map[types.UID]ConfigObject)
	for i := 0; i < childCount; i++ {
		result := <-resultsChan
		if result.err != nil {
			errs = append(errs, result.err.Error())
		}
		if result.obj != nil {
			if knownChild, exists := childMap[result.obj.GetUID()]; exists {
				if result.singleField {
					// If the known child only has single fields then just append the new single field,
					// otherwise the whole object is already being used.
					if knownChild.singleFields {
						knownChild.fieldKeys[result.fieldKey] = ConfigField{optional: result.fieldOptional}
					}
				} else {
					// Pulling in the whole object always overrides any use of single fields.
					knownChild.singleFields = false
					knownChild.fieldKeys = map[string]ConfigField{}
				}
			} else {
				childMap[result.obj.GetUID()] = ConfigObject{k8sObject: result.obj, singleFields: result.singleField, fieldKeys: map[string]ConfigField{}}
				if result.singleField {
					childMap[result.obj.GetUID()].fieldKeys[result.fieldKey] = ConfigField{optional: result.fieldOptional}
				}
			}
		}
	}

	// If there were any errors, don't return any children
	if len(errs) > 0 {
		return []ConfigObject{}, fmt.Errorf("error(s) encountered when geting children: %s", strings.Join(errs, ", "))
	}

	// Convert the map of children into an array
	var children []ConfigObject
	for _, child := range childMap {
		children = append(children, child)
	}

	// No errors, return the list of children
	return children, nil
}

// getChildNamesByType parses the Deployment object and returns four sets,
// the first containing the names of all referenced ConfigMaps,
// the second containing the names of all referenced Secrets,
// the third containing the name/key pairs of all referenced keys in a Config Map
// the forth containing the name/key pairs of all referenced keys in a Secret
func getChildNamesByType(obj *appsv1.Deployment) (map[string]struct{}, map[string]struct{}, map[string]map[string]ConfigField, map[string]map[string]ConfigField) {
	// Create sets for storing the names fo the ConfigMaps/Secrets
	configMaps := make(map[string]struct{})
	secrets := make(map[string]struct{})
	configMapKeyReferences := make(map[string]map[string]ConfigField)
	secretKeyReferences := make(map[string]map[string]ConfigField)

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

	// Range through all Containers and their respective Env
	for _, container := range obj.Spec.Template.Spec.Containers {
		for _, env := range container.Env {
			if valFrom := env.ValueFrom; valFrom != nil {
				if cm := valFrom.ConfigMapKeyRef; cm != nil {
					if configMapKeyReferences[cm.Name] == nil {
						configMapKeyReferences[cm.Name] = map[string]ConfigField{cm.Key: {optional: false}}
					}
					if cm.Optional != nil {
						configMapKeyReferences[cm.Name][cm.Key] = ConfigField{optional: *cm.Optional}
					} else {
						configMapKeyReferences[cm.Name][cm.Key] = ConfigField{optional: false}
					}
				}
				if s := valFrom.SecretKeyRef; s != nil {
					if secretKeyReferences[s.Name] == nil {
						secretKeyReferences[s.Name] = map[string]ConfigField{s.Key: {optional: false}}
					}
					if s.Optional != nil {
						secretKeyReferences[s.Name][s.Key] = ConfigField{optional: *s.Optional}
					} else {
						secretKeyReferences[s.Name][s.Key] = ConfigField{optional: false}
					}
				}
			}
		}
	}

	return configMaps, secrets, configMapKeyReferences, secretKeyReferences
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

// getConfigMap gets a ConfigMap with the given name and namespace from the
// API server.
func (h *Handler) getConfigMapWithKey(namespace, name, key string, optional bool) getResult {
	return h.getObjectWithKey(namespace, name, key, optional, &corev1.ConfigMap{})
}

// getSecret gets a Secret with the given name and namespace from the
// API server.
func (h *Handler) getSecretWithKey(namespace, name, key string, optional bool) getResult {
	return h.getObjectWithKey(namespace, name, key, optional, &corev1.Secret{})
}

// getObject gets the Object with the given name and namespace from the API
// server
func (h *Handler) getObject(namespace, name string, obj Object) getResult {
	objectName := types.NamespacedName{Namespace: namespace, Name: name}
	err := h.Get(context.TODO(), objectName, obj)
	if err != nil {
		return getResult{err: err}
	}
	return getResult{obj: obj, singleField: false}
}

// getObject gets the Object with the given name and namespace from the API
// server, and records the specific key requested
func (h *Handler) getObjectWithKey(namespace, name, key string, optional bool, obj Object) getResult {
	objectName := types.NamespacedName{Namespace: namespace, Name: name}
	err := h.Get(context.TODO(), objectName, obj)
	if err != nil {
		return getResult{err: err}
	}
	return getResult{obj: obj, singleField: true, fieldKey: key, fieldOptional: optional}
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
	// Deployment to the output list children
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
