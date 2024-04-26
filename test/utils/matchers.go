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
	"context"

	"github.com/onsi/gomega"
	gtypes "github.com/onsi/gomega/types"
	appsv1 "k8s.io/api/apps/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Matcher has Gomega Matchers that use the controller-runtime client
type Matcher struct {
	Client client.Client
}

// UpdateFunc modifies the object fetched from the API server before sending
// the update
type UpdateFunc func(client.Object) client.Object

// Create creates the object on the API server
func (m *Matcher) Create(obj client.Object, extras ...interface{}) gomega.GomegaAssertion {
	err := m.Client.Create(context.TODO(), obj)
	return gomega.Expect(err, extras)
}

// Delete deletes the object from the API server
func (m *Matcher) Delete(obj client.Object, extras ...interface{}) gomega.GomegaAssertion {
	err := m.Client.Delete(context.TODO(), obj)
	return gomega.Expect(err, extras)
}

// Update updates the object on the API server by fetching the object
// and applying a mutating UpdateFunc before sending the update
func (m *Matcher) Update(obj client.Object, fn UpdateFunc, intervals ...interface{}) gomega.GomegaAsyncAssertion {
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
func (m *Matcher) Get(obj client.Object, intervals ...interface{}) gomega.GomegaAsyncAssertion {
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
func (m *Matcher) Consistently(obj client.Object, intervals ...interface{}) gomega.GomegaAsyncAssertion {
	return m.consistentlyObject(obj, intervals...)
}

// consistentlyObject gets an individual object from the API server
func (m *Matcher) consistentlyObject(obj client.Object, intervals ...interface{}) gomega.GomegaAsyncAssertion {
	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
	get := func() client.Object {
		err := m.Client.Get(context.TODO(), key, obj)
		if err != nil {
			panic(err)
		}
		return obj
	}
	return gomega.Consistently(get, intervals...)
}

// WithAnnotations returns the object's Annotations
func WithAnnotations(matcher gtypes.GomegaMatcher) gtypes.GomegaMatcher {
	return gomega.WithTransform(func(obj client.Object) map[string]string {
		return obj.GetAnnotations()
	}, matcher)
}

// WithFinalizers returns the object's Finalizers
func WithFinalizers(matcher gtypes.GomegaMatcher) gtypes.GomegaMatcher {
	return gomega.WithTransform(func(obj client.Object) []string {
		return obj.GetFinalizers()
	}, matcher)
}

// WithItems returns the lists Finalizers
func WithItems(matcher gtypes.GomegaMatcher) gtypes.GomegaMatcher {
	return gomega.WithTransform(func(obj client.ObjectList) []client.Object {
		items, err := meta.ExtractList(obj)
		if err != nil {
			panic(err)
		}
		var objs []client.Object
		for _, item := range items {
			co, ok := item.(client.Object)
			if !ok {
				panic("Item is not a client.Object")
			}
			objs = append(objs, co)
		}
		return objs
	}, matcher)
}

// WithOwnerReferences returns the object's OwnerReferences
func WithOwnerReferences(matcher gtypes.GomegaMatcher) gtypes.GomegaMatcher {
	return gomega.WithTransform(func(obj client.Object) []metav1.OwnerReference {
		return obj.GetOwnerReferences()
	}, matcher)
}

// WithPodTemplateAnnotations returns the PodTemplate's annotations
func WithPodTemplateAnnotations(matcher gtypes.GomegaMatcher) gtypes.GomegaMatcher {
	return gomega.WithTransform(func(obj client.Object) map[string]string {
		switch obj := obj.(type) {
		case *appsv1.Deployment:
			return obj.Spec.Template.GetAnnotations()
		case *appsv1.StatefulSet:
			return obj.Spec.Template.GetAnnotations()
		case *appsv1.DaemonSet:
			return obj.Spec.Template.GetAnnotations()
		default:
			panic("Unknown pod template type.")
		}
	}, matcher)
}

// WithDeletionTimestamp returns the objects Deletion Timestamp
func WithDeletionTimestamp(matcher gtypes.GomegaMatcher) gtypes.GomegaMatcher {
	return gomega.WithTransform(func(obj client.Object) *metav1.Time {
		return obj.GetDeletionTimestamp()
	}, matcher)
}
