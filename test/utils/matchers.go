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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

// Update udpates the object on the API server
func (m *Matcher) Update(obj Object, extras ...interface{}) gomega.GomegaAssertion {
	err := m.Client.Update(context.TODO(), obj)
	return gomega.Expect(err, extras)
}
