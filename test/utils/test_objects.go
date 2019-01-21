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

package utils

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var labels = map[string]string{
	"app": "example",
}

// ExampleDeployment is an example Deployment object for use within test suites
var ExampleDeployment = &appsv1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "example",
		Namespace: "default",
		Labels:    labels,
	},
	Spec: appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "secret1",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "example1",
							},
						},
					},
					{
						Name: "configmap1",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "example1",
								},
							},
						},
					},
				},
				Containers: []corev1.Container{
					{
						Name:  "container1",
						Image: "container1",
						EnvFrom: []corev1.EnvFromSource{
							{
								ConfigMapRef: &corev1.ConfigMapEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "example1",
									},
								},
							},
							{
								SecretRef: &corev1.SecretEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "example1",
									},
								},
							},
						},
					},
					{
						Name:  "container2",
						Image: "container2",
						EnvFrom: []corev1.EnvFromSource{
							{
								ConfigMapRef: &corev1.ConfigMapEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "example2",
									},
								},
							},
							{
								SecretRef: &corev1.SecretEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "example2",
									},
								},
							},
						},
					},
				},
			},
		},
	},
}

// ExampleConfigMap1 is an example ConfigMap object for use within test suites
var ExampleConfigMap1 = &corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "example1",
		Namespace: "default",
		Labels:    labels,
	},
	Data: map[string]string{
		"key1": "example1:key1",
		"key2": "example1:key2",
		"key3": "example1:key3",
	},
}

// ExampleConfigMap2 is an example ConfigMap object for use within test suites
var ExampleConfigMap2 = &corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "example2",
		Namespace: "default",
		Labels:    labels,
	},
	Data: map[string]string{
		"key1": "example2:key1",
		"key2": "example2:key2",
		"key3": "example2:key3",
	},
}

// ExampleSecret1 is an example Secret object for use within test suites
var ExampleSecret1 = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "example1",
		Namespace: "default",
		Labels:    labels,
	},
	StringData: map[string]string{
		"key1": "example1:key1",
		"key2": "example1:key2",
		"key3": "example1:key3",
	},
}

// ExampleSecret2 is an example Secret object for use within test suites
var ExampleSecret2 = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "example2",
		Namespace: "default",
		Labels:    labels,
	},
	StringData: map[string]string{
		"key1": "example2:key1",
		"key2": "example2:key2",
		"key3": "example2:key3",
	},
}
