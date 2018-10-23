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

const requiredAnnotation = "wave.pusher.com/update-on-config-change"

// hasRequiredAnnotation returns true if the given Deployment has the wave
// annotation present
func hasRequiredAnnotation(obj *appsv1.Deployment) bool {
	annotations := obj.GetAnnotations()
	if value, ok := annotations[requiredAnnotation]; ok {
		if value == "true" {
			return true
		}
	}
	return false
}
