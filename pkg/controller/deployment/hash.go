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
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
)

const configHashAnnotation = "wave.pusher.com/config-hash"

// calculateConfigHash uses sha256 to hash the configuration within the child
// objects and returns a hash as a string
func calculateConfigHash(children []object) (string, error) {
	// TODO: implement this

	// TODO: remove this print: This is so the linter doesn't complain while this
	// method isn't implemented
	fmt.Printf("Config Hash Annotation: %s", configHashAnnotation)
	return "", nil
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
