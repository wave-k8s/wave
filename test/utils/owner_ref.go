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
