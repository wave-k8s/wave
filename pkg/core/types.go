package core

import (
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ConfigHashAnnotation is the key of the annotation on the PodTemplate that
	// holds the configuration hash
	ConfigHashAnnotation = "wave.pusher.com/config-hash"

	// FinalizerString is the finalizer added to deployments to allow Wave to
	// perform advanced deletion logic
	FinalizerString = "wave.pusher.com/finalizer"

	// SchedulingDisabledAnnotation is set on a deployment if scheduling has been disabled
	// due to missing children and contains the original scheduler
	SchedulingDisabledAnnotation = "wave.pusher.com/scheduling-disabled"

	// SchedulingDisabledSchedulerName is the dummy scheduler to disable scheduling of pods
	SchedulingDisabledSchedulerName = "wave.pusher.com/invalid"

	// ExtraConfigMapsAnnotation is the key of the annotation that contains additional
	// ConfigMaps which Wave should watch
	ExtraConfigMapsAnnotation = "wave.pusher.com/extra-configmaps"

	// ExtraSecretsAnnotation is the key of the annotation that contains additional
	// Secrets which Wave should watch
	ExtraSecretsAnnotation = "wave.pusher.com/extra-secrets"

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

type InstanceType interface {
	*appsv1.Deployment | *appsv1.StatefulSet | *appsv1.DaemonSet
	client.Object
	runtime.Object
	metav1.Object
}

type DeplyomentInterface interface {
	metav1.TypeMeta
	metav1.ObjectMeta
}

func GetPodTemplate[I InstanceType](instance I) *corev1.PodTemplateSpec {
	if deployment, ok := any(instance).(*appsv1.Deployment); ok {
		return &deployment.Spec.Template
	}
	if statefulset, ok := any(instance).(*appsv1.StatefulSet); ok {
		return &statefulset.Spec.Template
	}
	if daemonset, ok := any(instance).(*appsv1.DaemonSet); ok {
		return &daemonset.Spec.Template
	}
	panic(fmt.Sprintf("Invalid type %s", reflect.TypeOf(instance)))
}

func SetPodTemplate[I InstanceType](instance I, template *corev1.PodTemplateSpec) {
	if deployment, ok := any(instance).(*appsv1.Deployment); ok {
		deployment.Spec.Template = *template
	} else if statefulset, ok := any(instance).(*appsv1.StatefulSet); ok {
		statefulset.Spec.Template = *template
	} else if daemonset, ok := any(instance).(*appsv1.DaemonSet); ok {
		daemonset.Spec.Template = *template
	} else {
		panic(fmt.Sprintf("Invalid type %s", reflect.TypeOf(instance)))
	}
}

func GetNamespacedName(name string, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
}

type ObjectWithNameAndNamespace interface {
	GetNamespace() string
	GetName() string
}

func GetNamespacedNameFromObject(obj ObjectWithNameAndNamespace) types.NamespacedName {
	return types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
}
