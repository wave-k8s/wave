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
	"crypto/sha256"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// calculateConfigHash uses sha256 to hash the configuration within the child
// objects and returns a hash as a string
func calculateConfigHash(configMaps map[types.NamespacedName]*corev1.ConfigMap, secrets map[types.NamespacedName]*corev1.Secret, configMapsConfig configMetadataList, secretsConfig configMetadataList) (string, error) {
	// hashSource contains all the data to be hashed
	hashSource := struct {
		ConfigMaps map[string]map[string][]byte `json:"configMaps"`
		Secrets    map[string]map[string][]byte `json:"secrets"`
	}{
		ConfigMaps: make(map[string]map[string][]byte),
		Secrets:    make(map[string]map[string][]byte),
	}

	// Add the data from each child to the hashSource
	// All children should be in the same namespace so each one should have a
	// unique name
	for _, childConfig := range configMapsConfig {
		cm, ok := configMaps[childConfig.name]
		if !ok {
			continue
		}
		if _, ok := hashSource.ConfigMaps[childConfig.name.Name]; !ok {
			hashSource.ConfigMaps[childConfig.name.Name] = make(map[string][]byte)
		}
		hashSource.ConfigMaps[childConfig.name.Name] = addConfigMapData(hashSource.ConfigMaps[childConfig.name.Name], childConfig, cm)
	}

	for _, childConfig := range secretsConfig {
		s, ok := secrets[childConfig.name]
		if !ok {
			continue
		}
		if _, ok := hashSource.Secrets[childConfig.name.Name]; !ok {
			hashSource.Secrets[childConfig.name.Name] = make(map[string][]byte)
		}
		hashSource.Secrets[childConfig.name.Name] = addSecretData(hashSource.Secrets[childConfig.name.Name], childConfig, s)
	}

	// Convert the hashSource to a byte slice so that it can be hashed
	hashSourceBytes, err := json.Marshal(hashSource)
	if err != nil {
		return "", fmt.Errorf("unable to marshal JSON: %v", err)
	}

	hashBytes := sha256.Sum256(hashSourceBytes)
	return fmt.Sprintf("%x", hashBytes), nil
}

// addConfigMapData extracts all the relevant data from the ConfigMap, whether that is
// the whole ConfigMap or only the specified keys.
func addConfigMapData(data map[string][]byte, childConfig configMetadata, cm *corev1.ConfigMap) map[string][]byte {
	if childConfig.allKeys {
		for key := range cm.Data {
			data[key] = []byte(cm.Data[key])
		}
		for key, value := range cm.BinaryData {
			data[key] = value
		}
		return data
	}
	for key := range childConfig.keys {
		if value, exists := cm.Data[key]; exists {
			data[key] = []byte(value)
		}
		if value, exists := cm.BinaryData[key]; exists {
			data[key] = value
		}
	}
	return data
}

// getSecretData extracts all the relevant data from the Secret, whether that is
// the whole Secret or only the specified keys.
func addSecretData(data map[string][]byte, childConfig configMetadata, s *corev1.Secret) map[string][]byte {
	if childConfig.allKeys {
		for key, value := range s.Data {
			data[key] = value
		}
		return data
	}
	for key := range childConfig.keys {
		if value, exists := s.Data[key]; exists {
			data[key] = value
		}
	}
	return data
}

// setConfigHash updates the configuration hash of the given Deployment to the
// given string
func setConfigHash[I InstanceType](obj I, hash string) {
	// Get the existing annotations
	podTemplate := GetPodTemplate(obj)
	annotations := podTemplate.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Update the annotations
	annotations[ConfigHashAnnotation] = hash
	podTemplate.SetAnnotations(annotations)
	SetPodTemplate(obj, podTemplate)
}

// getConfigHash return the config hash string
func getConfigHash[I InstanceType](obj I) string {
	podTemplate := GetPodTemplate(obj)
	return podTemplate.GetAnnotations()[ConfigHashAnnotation]
}
