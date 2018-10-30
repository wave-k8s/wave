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

const finalizerString = "wave.pusher.com/finalizer"

// addFinalizer adds the wave finalizer to the given Deployment
func addFinalizer(obj *appsv1.Deployment) {
	finalizers := obj.GetFinalizers()
	for _, finalizer := range finalizers {
		if finalizer == finalizerString {
			// Deployment already contains the finalizer
			return
		}
	}

	//Deployment doens't contain the finalizer, so add it
	finalizers = append(finalizers, finalizerString)
	obj.SetFinalizers(finalizers)
}

// removeFinalizer removes the wave finalizer from the given Deployment
func removeFinalizer(obj *appsv1.Deployment) {
	finalizers := obj.GetFinalizers()

	// Filter existing finalizers removing any that match the finalizerString
	newFinalizers := []string{}
	for _, finalizer := range finalizers {
		if finalizer != finalizerString {
			newFinalizers = append(newFinalizers, finalizer)
		}
	}

	// Update the object's finalizers
	obj.SetFinalizers(newFinalizers)
}
