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

package utils

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetOwnerRefDeployment constructs an owner reference for the Deployment given
func GetOwnerRefDeployment(deployment *appsv1.Deployment) metav1.OwnerReference {
	f := false
	t := true
	return metav1.OwnerReference{
		APIVersion:         "apps/v1",
		Kind:               "Deployment",
		Name:               deployment.Name,
		UID:                deployment.UID,
		Controller:         &f,
		BlockOwnerDeletion: &t,
	}
}

// GetOwnerRefStatefulSet constructs an owner reference for the StatefulSet given
func GetOwnerRefStatefulSet(sts *appsv1.StatefulSet) metav1.OwnerReference {
	f := false
	t := true
	return metav1.OwnerReference{
		APIVersion:         "apps/v1",
		Kind:               "StatefulSet",
		Name:               sts.Name,
		UID:                sts.UID,
		Controller:         &f,
		BlockOwnerDeletion: &t,
	}
}

// GetOwnerRefDaemonSet constructs an owner reference for the DaemonSet given
func GetOwnerRefDaemonSet(sts *appsv1.DaemonSet) metav1.OwnerReference {
	f := false
	t := true
	return metav1.OwnerReference{
		APIVersion:         "apps/v1",
		Kind:               "DaemonSet",
		Name:               sts.Name,
		UID:                sts.UID,
		Controller:         &f,
		BlockOwnerDeletion: &t,
	}
}
