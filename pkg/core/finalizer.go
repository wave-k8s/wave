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

// removeFinalizer removes the wave finalizer from the given podController
func removeFinalizer[I InstanceType](obj I) {
	finalizers := obj.GetFinalizers()

	// Filter existing finalizers removing any that match the finalizerString
	newFinalizers := []string{}
	for _, finalizer := range finalizers {
		if finalizer != FinalizerString {
			newFinalizers = append(newFinalizers, finalizer)
		}
	}

	// Update the object's finalizers
	obj.SetFinalizers(newFinalizers)
}

// hasFinalizer checks for the presence of the Wave finalizer
func hasFinalizer[I InstanceType](obj I) bool {
	finalizers := obj.GetFinalizers()
	for _, finalizer := range finalizers {
		if finalizer == FinalizerString {
			// podController already contains the finalizer
			return true
		}
	}

	return false
}
