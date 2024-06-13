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
	"context"
	"fmt"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getResult is returned from the getObject method as a helper struct to be
// passed into a channel
type getResult struct {
	Object
	types.NamespacedName
	error
}

// getCurrentChildren returns a list of all Secrets and ConfigMaps that are
// referenced in the Deployment's spec.  Any reference to a whole ConfigMap or Secret
// (i.e. via an EnvFrom or a Volume) will result in one entry in the list, irrespective of
// whether individual elements are also references (i.e. via an Env entry).
func (h *Handler[I]) getCurrentChildren(configMapsConfig configMetadataList, secretsConfig configMetadataList) (map[types.NamespacedName]*corev1.ConfigMap, map[types.NamespacedName]*corev1.Secret, error) {
	uniqueConfigMaps := make(map[types.NamespacedName]bool)
	uniqueSecrets := make(map[types.NamespacedName]bool)

	for _, cm := range configMapsConfig {
		uniqueConfigMaps[cm.name] = true
	}
	for _, s := range secretsConfig {
		uniqueSecrets[s.name] = true
	}

	// get all of ConfigMaps and Secrets
	resultsChan := make(chan getResult)
	for cm := range uniqueConfigMaps {
		go func(name types.NamespacedName) {
			resultsChan <- h.getConfigMap(name)
		}(cm)
	}
	for s := range uniqueSecrets {
		go func(name types.NamespacedName) {
			resultsChan <- h.getSecret(name)
		}(s)
	}

	// Range over and collect results from the gets
	var errs []string
	secrets := make(map[types.NamespacedName]*corev1.Secret)
	configMaps := make(map[types.NamespacedName]*corev1.ConfigMap)
	for i := 0; i < len(uniqueConfigMaps)+len(uniqueSecrets); i++ {
		result := <-resultsChan
		if result.error != nil {
			errs = append(errs, result.error.Error())
		}
		if result.Object != nil {
			if s, ok := result.Object.(*corev1.Secret); ok {
				secrets[result.NamespacedName] = s
			} else if cm, ok := result.Object.(*corev1.ConfigMap); ok {
				configMaps[result.NamespacedName] = cm
			} else {
				return nil, nil, fmt.Errorf("passed unknown type: %v", reflect.TypeOf(result.Object))
			}
		}
	}

	// If there were any errors, don't return any children
	if len(errs) > 0 {
		return nil, nil, fmt.Errorf("error(s) encountered when geting children: %s", strings.Join(errs, ", "))
	}

	// No errors, return the list of children
	return configMaps, secrets, nil
}

// getChildNamesByType parses the Deployment object and returns two maps,
// the first containing ConfigMap metadata for all referenced ConfigMaps, keyed on the name of the ConfigMap,
// the second containing Secret metadata for all referenced Secrets, keyed on the name of the Secrets
func getChildNamesByType[I InstanceType](obj I) (configMetadataList, configMetadataList) {
	// Create sets for storing the names fo the ConfigMaps/Secrets
	configMaps := configMetadataList{}
	secrets := configMetadataList{}

	// Range through all Volumes and check the VolumeSources for ConfigMaps
	// and Secrets
	for _, vol := range GetPodTemplate(obj).Spec.Volumes {
		if cm := vol.VolumeSource.ConfigMap; cm != nil {
			configMaps = append(configMaps, configMetadata{required: isRequired(cm.Optional), allKeys: true, name: GetNamespacedName(cm.Name, obj.GetNamespace())})
		}
		if s := vol.VolumeSource.Secret; s != nil {
			secrets = append(secrets, configMetadata{required: isRequired(s.Optional), allKeys: true, name: GetNamespacedName(s.SecretName, obj.GetNamespace())})
		}

		if projection := vol.VolumeSource.Projected; projection != nil {
			for _, source := range projection.Sources {
				if cm := source.ConfigMap; cm != nil {
					if cm.Items == nil {
						configMaps = append(configMaps, configMetadata{required: isRequired(cm.Optional), allKeys: true, name: GetNamespacedName(cm.Name, obj.GetNamespace())})
					} else {
						keys := make(map[string]struct{})
						for _, item := range cm.Items {
							keys[item.Key] = struct{}{}
						}
						configMaps = append(configMaps, configMetadata{required: isRequired(cm.Optional), allKeys: false, keys: keys, name: GetNamespacedName(cm.Name, obj.GetNamespace())})
					}
				}
				if s := source.Secret; s != nil {
					if s.Items == nil {
						secrets = append(secrets, configMetadata{required: isRequired(s.Optional), allKeys: true, name: GetNamespacedName(s.Name, obj.GetNamespace())})
					} else {
						keys := make(map[string]struct{})
						for _, item := range s.Items {
							keys[item.Key] = struct{}{}
						}
						secrets = append(secrets, configMetadata{required: isRequired(s.Optional), allKeys: false, keys: keys, name: GetNamespacedName(s.Name, obj.GetNamespace())})
					}
				}
			}
		}
	}

	// Parse deployment annotations for cms/secrets used inside the pod
	if annotations := obj.GetAnnotations(); annotations != nil {
		if configMapString, ok := annotations[ExtraConfigMapsAnnotation]; ok {
			for _, cm := range strings.Split(configMapString, ",") {
				parts := strings.Split(cm, "/")
				if len(parts) == 1 {
					configMaps = append(configMaps, configMetadata{required: false, allKeys: true, name: GetNamespacedName(parts[0], obj.GetNamespace())})
				} else if len(parts) == 2 {
					configMaps = append(configMaps, configMetadata{required: false, allKeys: true, name: GetNamespacedName(parts[1], parts[0])})
				}
			}
		}
		if secretString, ok := annotations[ExtraSecretsAnnotation]; ok {
			for _, secret := range strings.Split(secretString, ",") {
				parts := strings.Split(secret, "/")
				if len(parts) == 1 {
					secrets = append(secrets, configMetadata{required: false, allKeys: true, name: GetNamespacedName(parts[0], obj.GetNamespace())})
				} else if len(parts) == 2 {
					secrets = append(secrets, configMetadata{required: false, allKeys: true, name: GetNamespacedName(parts[1], parts[0])})
				}
			}
		}
	}

	// Range through all Containers and their respective EnvFrom,
	// then check the EnvFromSources for ConfigMaps and Secrets
	for _, container := range GetPodTemplate(obj).Spec.Containers {
		for _, env := range container.EnvFrom {
			if cm := env.ConfigMapRef; cm != nil {
				configMaps = append(configMaps, configMetadata{required: isRequired(cm.Optional), allKeys: true, name: GetNamespacedName(cm.Name, obj.GetNamespace())})
			}
			if s := env.SecretRef; s != nil {
				secrets = append(secrets, configMetadata{required: isRequired(s.Optional), allKeys: true, name: GetNamespacedName(s.Name, obj.GetNamespace())})
			}
		}
	}

	// Range through all Containers and their respective Env
	for _, container := range GetPodTemplate(obj).Spec.Containers {
		for _, env := range container.Env {
			if valFrom := env.ValueFrom; valFrom != nil {
				if cm := valFrom.ConfigMapKeyRef; cm != nil {
					keys := map[string]struct{}{
						cm.Key: {},
					}
					configMaps = append(configMaps, configMetadata{required: isRequired(cm.Optional), allKeys: false, keys: keys, name: GetNamespacedName(cm.Name, obj.GetNamespace())})
				}
				if s := valFrom.SecretKeyRef; s != nil {
					keys := map[string]struct{}{
						s.Key: {},
					}
					secrets = append(secrets, configMetadata{required: isRequired(s.Optional), allKeys: false, keys: keys, name: GetNamespacedName(s.Name, obj.GetNamespace())})
				}
			}
		}
	}

	return configMaps, secrets
}

func (h *Handler[I]) checkRequiredChildren(configMaps map[types.NamespacedName]*corev1.ConfigMap, secrets map[types.NamespacedName]*corev1.Secret, configMapsConfig configMetadataList, secretsConfig configMetadataList) error {
	errors := []string{}
	for _, childConfig := range configMapsConfig {
		if !childConfig.required {
			continue
		}
		cm, ok := configMaps[childConfig.name]
		if !ok {
			errors = append(errors, fmt.Sprintf("missing required configmap %s/%s", childConfig.name.Namespace, childConfig.name.Name))
			continue
		}
		if childConfig.keys != nil {
			for key := range childConfig.keys {
				_, inData := cm.Data[key]
				_, inBinaryData := cm.BinaryData[key]
				if !inData && !inBinaryData {
					errors = append(errors, fmt.Sprintf("missing required key %s in configmap %s/%s", key, childConfig.name.Namespace, childConfig.name.Name))
				}
			}
		}
	}

	for _, childConfig := range secretsConfig {
		if !childConfig.required {
			continue
		}
		s, ok := secrets[childConfig.name]
		if !ok {
			errors = append(errors, fmt.Sprintf("missing required secret %s/%s", childConfig.name.Namespace, childConfig.name.Name))
			continue
		}
		if childConfig.keys != nil {
			for key := range childConfig.keys {
				if _, ok := s.Data[key]; !ok {
					errors = append(errors, fmt.Sprintf("missing required key %s in secret %s/%s", key, childConfig.name.Namespace, childConfig.name.Name))
				}
			}
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("not all required children exist: %s", strings.Join(errors, ", "))
	}
	return nil
}

func isRequired(b *bool) bool {
	return b == nil || !*b
}

// getConfigMap gets a ConfigMap with the given name and namespace from the
// API server.
func (h *Handler[I]) getConfigMap(name types.NamespacedName) getResult {
	return h.getObject(name, &corev1.ConfigMap{})
}

// getSecret gets a Secret with the given name and namespace from the
// API server.
func (h *Handler[I]) getSecret(name types.NamespacedName) getResult {
	return h.getObject(name, &corev1.Secret{})
}

// getObject gets the Object with the given name and namespace from the API
// server
func (h *Handler[I]) getObject(name types.NamespacedName, obj Object) getResult {
	err := h.Get(context.TODO(), name, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			return getResult{nil, name, nil}
		}
		return getResult{nil, name, err}
	}
	return getResult{obj, name, nil}
}

// getExistingChildren returns a list of all Secrets and ConfigMaps that are
// owned by the Deployment instance
//
// Deprecated: Wave no longer uses OwnerReferences. Only used for migration.
func (h *Handler[I]) getExistingChildren(obj I) ([]Object, error) {
	inNamespace := client.InNamespace(obj.GetNamespace())

	// List all ConfigMaps in the Deployment's namespace
	configMaps := &corev1.ConfigMapList{}
	err := h.List(context.TODO(), configMaps, inNamespace)
	if err != nil {
		return []Object{}, fmt.Errorf("error listing ConfigMaps: %v", err)
	}

	// List all Secrets in the Deployment's namespcae
	secrets := &corev1.SecretList{}
	err = h.List(context.TODO(), secrets, inNamespace)
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
//
// Deprecated: Wave no longer uses OwnerReferences. Only used for migration.
func isOwnedBy(child, owner metav1.Object) bool {
	for _, ref := range child.GetOwnerReferences() {
		if ref.UID == owner.GetUID() {
			return true
		}
	}
	return false
}
