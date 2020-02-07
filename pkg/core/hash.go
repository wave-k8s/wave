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

package core

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
)

// calculateConfigHash uses sha256 to hash the configuration within the child
// objects and returns a hash as a string
func calculateConfigHash(children []configObject) (string, error) {
	// hashSource contains all the data to be hashed
	hashSource := struct {
		ConfigMaps map[string]map[string]string `json:"configMaps"`
		Secrets    map[string]map[string][]byte `json:"secrets"`
	}{
		ConfigMaps: make(map[string]map[string]string),
		Secrets:    make(map[string]map[string][]byte),
	}

	// Add the data from each child to the hashSource
	// All children should be in the same namespace so each one should have a
	// unique name
	for _, child := range children {
		if child.object != nil {
			switch child.object.(type) {
			case *corev1.ConfigMap:
				hashSource.ConfigMaps[child.object.GetName()] = getConfigMapData(child)
			case *corev1.Secret:
				hashSource.Secrets[child.object.GetName()] = getSecretData(child)
			default:
				return "", fmt.Errorf("passed unknown type: %v", reflect.TypeOf(child))
			}
		}
	}

	// Convert the hashSource to a byte slice so that it can be hashed
	hashSourceBytes, err := json.Marshal(hashSource)
	if err != nil {
		return "", fmt.Errorf("unable to marshal JSON: %v", err)
	}

	hashBytes := sha256.Sum256(hashSourceBytes)
	return fmt.Sprintf("%x", hashBytes), nil
}

// updateHashBank stores hashes of each child in the in memory hashbank. It is
// not threadsafe, and not to be called other than by a helper function.  as the
// children should be uniquely named within the namespace it does not store the
// instance name.
func (h *Handler) updateHashBank(children []configObject) error {
	// Add the hashes of each childs data to the hashBank
	for _, child := range children {
		if child.object != nil {
			switch child.object.(type) {
			case *corev1.ConfigMap:
				hash, err := hashData(getConfigMapData(child))
				if err != nil {
					// log event and continue?
					// return the error?
				}
				h.store.Bank[child.object.GetName()] = hashEntry{hash}
			case *corev1.Secret:
				hash, err := hashData(getSecretData(child))
				if err != nil {
					// log event and continue?
				}
				h.store.Bank[child.object.GetName()] = hashEntry{hash}
			default:
				return fmt.Errorf("passed unknown type: %v", reflect.TypeOf(child))
			}
		}
	}
	return nil
}

// initialiseHashBank checks if the instanceName exists in the hashBank, if not
// it calls updateHashBank on the children.
func (h *Handler) initialiseHashBank(instanceName string, children []configObject) error {
	h.store.Lock()
	defer h.store.Unlock()
	if h.store.Bank == nil {
		h.store.Bank = make(map[string]hashEntry)
	}
	if h.store.initialised == nil {
		h.store.initialised = make(map[string]bool)
	}

	if _, ok := h.store.initialised[instanceName]; !ok {
		err := h.updateHashBank(children)
		if err != nil {
			return err
		}
		h.store.initialised[instanceName] = true
	}
	return nil
}

// retrieveFromHashBank returns a map of objectName:hash for all children provided
func (h *Handler) retrieveFromHashBank(children []configObject) (map[string]string, error) {
	h.store.RLock()
	defer h.store.RUnlock()
	output := make(map[string]string)

	// Retrieve the hashes of each childs data to the hashBank
	for _, child := range children {
		if child.object != nil {
			switch child.object.(type) {
			case *corev1.ConfigMap:
				shaEntry := h.store.Bank[child.object.GetName()]
				output[child.object.GetName()] = shaEntry.entry
			case *corev1.Secret:
				shaEntry := h.store.Bank[child.object.GetName()]
				output[child.object.GetName()] = shaEntry.entry
			default:
				return output, fmt.Errorf("passed unknown type: %v", reflect.TypeOf(child))
			}
		}
	}
	return output, nil

}

// calculateNewHashBankEntries updates the hashbank given a new set of children
func (h *Handler) calculateNewHashBankEntries(instanceName string, children []configObject) error {
	h.store.Lock()
	defer h.store.Unlock()
	if _, ok := h.store.initialised[instanceName]; ok {
		err := h.updateHashBank(children)
		if err != nil {
			return err
		}
	}
	return nil
}

// getConfigMapData extracts all the relevant data from the ConfigMap, whether that is
// the whole ConfigMap or only the specified keys.
func getConfigMapData(child configObject) map[string]string {
	cm := *child.object.(*corev1.ConfigMap)
	if child.allKeys {
		return cm.Data
	}
	keyData := make(map[string]string)
	for key := range child.keys {
		if value, exists := cm.Data[key]; exists {
			keyData[key] = value
		}
	}
	return keyData
}

// getSecretData extracts all the relevant data from the Secret, whether that is
// the whole Secret or only the specified keys.
func getSecretData(child configObject) map[string][]byte {
	s := *child.object.(*corev1.Secret)
	if child.allKeys {
		return s.Data
	}
	keyData := make(map[string][]byte)
	for key := range child.keys {
		if value, exists := s.Data[key]; exists {
			keyData[key] = value
		}
	}
	return keyData
}

// setConfigHash upates the configuration hash of the given Deployment to the
// given string
func setConfigHash(obj podController, hash string) {
	// Get the existing annotations
	podTemplate := obj.GetPodTemplate()
	annotations := podTemplate.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Update the annotations
	annotations[ConfigHashAnnotation] = hash
	podTemplate.SetAnnotations(annotations)
	obj.SetPodTemplate(podTemplate)
}

// hashData takes input and returns sha256'd output.
func hashData(data interface{}) (string, error) {
	structured, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("unable to marshal JSON: %v", err)
	}

	hashBytes := sha256.Sum256(structured)
	return fmt.Sprintf("%x", hashBytes), nil
}
