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
	"context"

	"github.com/onsi/gomega"
	gtypes "github.com/onsi/gomega/types"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Matcher has Gomega Matchers that use the controller-runtime client
type Matcher struct {
	Client client.Client
}

// Object is the combination of two interfaces as a helper for passing
// Kubernetes objects between methods
type Object interface {
	runtime.Object
	metav1.Object
}

// UpdateFunc modifies the object fetched from the API server before sending
// the update
type UpdateFunc func(Object) Object

// Create creates the object on the API server
func (m *Matcher) Create(obj Object, extras ...interface{}) gomega.GomegaAssertion {
	err := m.Client.Create(context.TODO(), obj)
	return gomega.Expect(err, extras)
}

// Delete deletes the object from the API server
func (m *Matcher) Delete(obj Object, extras ...interface{}) gomega.GomegaAssertion {
	err := m.Client.Delete(context.TODO(), obj)
	return gomega.Expect(err, extras)
}

// UpdateWithFunc udpates the object on the API server by fetching the object
// and applying a mutating UpdateFunc before sending the update
func (m *Matcher) UpdateWithFunc(obj Object, fn UpdateFunc, intervals ...interface{}) gomega.GomegaAsyncAssertion {
	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
	update := func() error {
		err := m.Client.Get(context.TODO(), key, obj)
		if err != nil {
			return err
		}
		return m.Client.Update(context.TODO(), fn(obj))
	}
	return gomega.Eventually(update, intervals...)
}

// Get gets the object from the API server
func (m *Matcher) Get(obj Object, intervals ...interface{}) gomega.GomegaAsyncAssertion {
	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
	get := func() error {
		return m.Client.Get(context.TODO(), key, obj)
	}
	return gomega.Eventually(get, intervals...)
}

// Consistently continually gets the object from the API for comparison
func (m *Matcher) Consistently(obj Object, intervals ...interface{}) gomega.GomegaAsyncAssertion {
	return m.consistentlyObject(obj, intervals...)
}

// consistentlyObject gets an individual object from the API server
func (m *Matcher) consistentlyObject(obj Object, intervals ...interface{}) gomega.GomegaAsyncAssertion {
	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
	get := func() Object {
		err := m.Client.Get(context.TODO(), key, obj)
		if err != nil {
			panic(err)
		}
		return obj
	}
	return gomega.Consistently(get, intervals...)
}

// Eventually continually gets the object from the API for comparison
func (m *Matcher) Eventually(obj runtime.Object, intervals ...interface{}) gomega.GomegaAsyncAssertion {
	// If the object is a list, return a list
	if meta.IsListType(obj) {
		return m.eventuallyList(obj, intervals...)
	}
	if o, ok := obj.(Object); ok {
		return m.eventuallyObject(o, intervals...)
	}
	//Should not get here
	panic("Unknown object.")
}

// eventuallyObject gets an individual object from the API server
func (m *Matcher) eventuallyObject(obj Object, intervals ...interface{}) gomega.GomegaAsyncAssertion {
	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
	get := func() Object {
		err := m.Client.Get(context.TODO(), key, obj)
		if err != nil {
			panic(err)
		}
		return obj
	}
	return gomega.Eventually(get, intervals...)
}

// eventuallyList gets a list type  from the API server
func (m *Matcher) eventuallyList(obj runtime.Object, intervals ...interface{}) gomega.GomegaAsyncAssertion {
	list := func() runtime.Object {
		err := m.Client.List(context.TODO(), obj)
		if err != nil {
			panic(err)
		}
		return obj
	}
	return gomega.Eventually(list, intervals...)
}

// WithAnnotations returns the object's Annotations
func WithAnnotations(matcher gtypes.GomegaMatcher) gtypes.GomegaMatcher {
	return gomega.WithTransform(func(obj Object) map[string]string {
		return obj.GetAnnotations()
	}, matcher)
}

// WithFinalizers returns the object's Finalizers
func WithFinalizers(matcher gtypes.GomegaMatcher) gtypes.GomegaMatcher {
	return gomega.WithTransform(func(obj Object) []string {
		return obj.GetFinalizers()
	}, matcher)
}

// WithItems returns the lists Finalizers
func WithItems(matcher gtypes.GomegaMatcher) gtypes.GomegaMatcher {
	return gomega.WithTransform(func(obj runtime.Object) []runtime.Object {
		items, err := meta.ExtractList(obj)
		if err != nil {
			panic(err)
		}
		return items
	}, matcher)
}

// WithOwnerReferences returns the object's OwnerReferences
func WithOwnerReferences(matcher gtypes.GomegaMatcher) gtypes.GomegaMatcher {
	return gomega.WithTransform(func(obj Object) []metav1.OwnerReference {
		return obj.GetOwnerReferences()
	}, matcher)
}

// WithPodTemplateAnnotations returns the PodTemplate's annotations
func WithPodTemplateAnnotations(matcher gtypes.GomegaMatcher) gtypes.GomegaMatcher {
	return gomega.WithTransform(func(obj Object) map[string]string {
		switch obj.(type) {
		case *appsv1.Deployment:
			return obj.(*appsv1.Deployment).Spec.Template.GetAnnotations()
		case *appsv1.StatefulSet:
			return obj.(*appsv1.StatefulSet).Spec.Template.GetAnnotations()
		case *appsv1.DaemonSet:
			return obj.(*appsv1.DaemonSet).Spec.Template.GetAnnotations()
		default:
			panic("Unknown pod template type.")
		}
	}, matcher)
}

// WithDeletionTimestamp returns the objects Deletion Timestamp
func WithDeletionTimestamp(matcher gtypes.GomegaMatcher) gtypes.GomegaMatcher {
	return gomega.WithTransform(func(obj Object) *metav1.Time {
		return obj.GetDeletionTimestamp()
	}, matcher)
}
