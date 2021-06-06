package core

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// ConfigHashAnnotation is the key of the annotation on the PodTemplate that
	// holds the configuratio hash
	ConfigHashAnnotation = "wave.pusher.com/config-hash"

	// FinalizerString is the finalizer added to deployments to allow Wave to
	// perform advanced deletion logic
	FinalizerString = "wave.pusher.com/finalizer"

	// RequiredAnnotation is the key of the annotation on the Deployment that Wave
	// checks for before processing the deployment
	RequiredAnnotation = "wave.pusher.com/update-on-config-change"

	// requiredAnnotationValue is the value of the annotation on the Deployment that Wave
	// checks for before processing the deployment
	requiredAnnotationValue = "true"
)

// Object is used as a helper interface when passing Kubernetes resources
// between methods.
// All Kubernetes resources should implement both of these interfaces
type Object interface {
	runtime.Object
	metav1.Object
}

// configObject is used as a container of an "Object" along with metadata
// that Wave uses to determine what to use from that Object.
type configObject struct {
	object   Object
	required bool
	allKeys  bool
	keys     map[string]struct{}
}

type podController interface {
	runtime.Object
	metav1.Object
	GetObject() runtime.Object
	GetPodTemplate() *corev1.PodTemplateSpec
	SetPodTemplate(*corev1.PodTemplateSpec)
	DeepCopy() podController
}

type deployment struct {
	*appsv1.Deployment
}

func (d *deployment) GetObject() runtime.Object {
	return d.Deployment
}

func (d *deployment) GetPodTemplate() *corev1.PodTemplateSpec {
	return &d.Deployment.Spec.Template
}

func (d *deployment) SetPodTemplate(template *corev1.PodTemplateSpec) {
	d.Deployment.Spec.Template = *template
}

func (d *deployment) DeepCopy() podController {
	return &deployment{d.Deployment.DeepCopy()}
}

type statefulset struct {
	*appsv1.StatefulSet
}

func (d *statefulset) GetObject() runtime.Object {
	return d.StatefulSet
}

func (d *statefulset) GetPodTemplate() *corev1.PodTemplateSpec {
	return &d.StatefulSet.Spec.Template
}

func (d *statefulset) SetPodTemplate(template *corev1.PodTemplateSpec) {
	d.StatefulSet.Spec.Template = *template
}

func (d *statefulset) DeepCopy() podController {
	return &statefulset{d.StatefulSet.DeepCopy()}
}

type daemonset struct {
	*appsv1.DaemonSet
}

func (d *daemonset) GetObject() runtime.Object {
	return d.DaemonSet
}

func (d *daemonset) GetPodTemplate() *corev1.PodTemplateSpec {
	return &d.DaemonSet.Spec.Template
}

func (d *daemonset) SetPodTemplate(template *corev1.PodTemplateSpec) {
	d.DaemonSet.Spec.Template = *template
}

func (d *daemonset) DeepCopy() podController {
	return &daemonset{d.DaemonSet.DeepCopy()}
}
