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
	"crypto/sha256"
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
)

const configHashAnnotation = "wave.pusher.com/config-hash"

// calculateConfigHash uses sha256 to hash the configuration within the child
// objects and returns a hash as a string
func calculateConfigHash(children []object) (string, error) {
	// hashSource contains all the data to be hashed
	hashSource := struct {
		ConfigMaps map[string]map[string]string `json:"configMaps"`
		Secrets    map[string]map[string][]byte `json:"secrets"`
	}{
		ConfigMaps: make(map[string]map[string]string),
		Secrets:    make(map[string]map[string][]byte),
	}

	// Convert the hashSource to a byte slice so that it can be hashed
	hashSourceBytes, err := json.Marshal(hashSource)
	if err != nil {
		return "", fmt.Errorf("unable to marshal JSON: %v", err)
	}

	hashBytes := sha256.Sum256(hashSourceBytes)
	return fmt.Sprintf("%x", hashBytes), nil
}

// setConfigHash upates the configuration hash of the given Deployment to the
// given string
func setConfigHash(obj *appsv1.Deployment, hash string) {
	// Get the existing annotations
	annotations := obj.Spec.Template.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Update the annotations
	annotations[configHashAnnotation] = hash
	obj.Spec.Template.SetAnnotations(annotations)
}
