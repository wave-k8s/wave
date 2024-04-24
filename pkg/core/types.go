package core

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ConfigHashAnnotation is the key of the annotation on the PodTemplate that
	// holds the configuration hash
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
// between methods. Adjusted to satisfy client.Object directly.
type Object interface {
	client.Object
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

// podController interface adjusted to include client.Object explicitly
type podController interface {
	client.Object
	metav1.Object
	GetPodTemplate() *corev1.PodTemplateSpec
	SetPodTemplate(*corev1.PodTemplateSpec)
	DeepCopyPodController() podController
	GetApiObject() client.Object
}

// Deployment struct implementing the podController interface
type deployment struct {
	*appsv1.Deployment
}

func (d *deployment) GetPodTemplate() *corev1.PodTemplateSpec {
	return &d.Spec.Template
}

func (d *deployment) SetPodTemplate(template *corev1.PodTemplateSpec) {
	d.Spec.Template = *template
}

func (d *deployment) DeepCopyPodController() podController {
	return &deployment{d.Deployment.DeepCopy()}
}

func (d *deployment) GetApiObject() client.Object {
	return &appsv1.Deployment{
		Status:     d.Status,
		Spec:       d.Spec,
		ObjectMeta: d.ObjectMeta,
	}
}

// StatefulSet struct implementing the podController interface
type statefulset struct {
	*appsv1.StatefulSet
}

func (s *statefulset) GetPodTemplate() *corev1.PodTemplateSpec {
	return &s.Spec.Template
}

func (s *statefulset) SetPodTemplate(template *corev1.PodTemplateSpec) {
	s.Spec.Template = *template
}

func (s *statefulset) DeepCopyPodController() podController {
	return &statefulset{s.StatefulSet.DeepCopy()}
}

func (d *statefulset) GetApiObject() client.Object {
	return &appsv1.StatefulSet{
		Status:     d.Status,
		Spec:       d.Spec,
		ObjectMeta: d.ObjectMeta,
	}
}

// DaemonSet struct implementing the podController interface
type daemonset struct {
	*appsv1.DaemonSet
}

func (d *daemonset) GetPodTemplate() *corev1.PodTemplateSpec {
	return &d.Spec.Template
}

func (d *daemonset) SetPodTemplate(template *corev1.PodTemplateSpec) {
	d.Spec.Template = *template
}

func (d *daemonset) DeepCopyPodController() podController {
	return &daemonset{d.DaemonSet.DeepCopy()}
}

func (d *daemonset) GetApiObject() client.Object {
	return &appsv1.DaemonSet{
		Status:     d.Status,
		Spec:       d.Spec,
		ObjectMeta: d.ObjectMeta,
	}
}
