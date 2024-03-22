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

package core

import (
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/wave-k8s/wave/test/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ = Describe("Wave hash Suite", func() {
	Context("calculateConfigHash", func() {
		var c client.Client
		var m utils.Matcher

		var mgrStopped *sync.WaitGroup
		var stopMgr chan struct{}

		const timeout = time.Second * 5

		var cm1 *corev1.ConfigMap
		var cm2 *corev1.ConfigMap
		var cm3 *corev1.ConfigMap
		var s1 *corev1.Secret
		var s2 *corev1.Secret
		var s3 *corev1.Secret

		var modified = "modified"

		BeforeEach(func() {
			mgr, err := manager.New(cfg, manager.Options{
				Metrics: metricsserver.Options{
					BindAddress: "0",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			var cerr error
			c, cerr = client.New(cfg, client.Options{Scheme: scheme.Scheme})
			Expect(cerr).NotTo(HaveOccurred())
			m = utils.Matcher{Client: c}

			stopMgr, mgrStopped = StartTestManager(mgr)

			cm1 = utils.ExampleConfigMap1.DeepCopy()
			cm2 = utils.ExampleConfigMap2.DeepCopy()
			cm3 = utils.ExampleConfigMap3.DeepCopy()
			s1 = utils.ExampleSecret1.DeepCopy()
			s2 = utils.ExampleSecret2.DeepCopy()
			s3 = utils.ExampleSecret3.DeepCopy()

			m.Create(cm1).Should(Succeed())
			m.Create(cm2).Should(Succeed())
			m.Create(cm3).Should(Succeed())
			m.Create(s1).Should(Succeed())
			m.Create(s2).Should(Succeed())
			m.Create(s3).Should(Succeed())

			m.Get(cm1, timeout).Should(Succeed())
			m.Get(cm2, timeout).Should(Succeed())
			m.Get(cm3, timeout).Should(Succeed())
			m.Get(s1, timeout).Should(Succeed())
			m.Get(s2, timeout).Should(Succeed())
			m.Get(s3, timeout).Should(Succeed())
		})

		AfterEach(func() {
			close(stopMgr)
			mgrStopped.Wait()

			utils.DeleteAll(cfg, timeout,
				&appsv1.DeploymentList{},
				&corev1.ConfigMapList{},
				&corev1.SecretList{},
			)
		})

		It("returns a different hash when an allKeys child's data is updated", func() {
			c := []configObject{
				{object: cm1, allKeys: true},
				{object: cm2, allKeys: true},
				{object: s1, allKeys: true},
				{object: s2, allKeys: true},
			}

			h1, err := calculateConfigHash(c)
			Expect(err).NotTo(HaveOccurred())

			m.Update(cm1, func(obj client.Object) client.Object {
				cm := obj.(*corev1.ConfigMap)
				cm.Data["key1"] = modified

				return cm
			}, timeout).Should(Succeed())
			h2, err := calculateConfigHash(c)
			Expect(err).NotTo(HaveOccurred())

			Expect(h2).NotTo(Equal(h1))
		})

		It("returns a different hash when an all-field child's data is updated", func() {
			c := []configObject{
				{object: cm1, allKeys: true},
				{object: cm2, allKeys: true},
				{object: s1, allKeys: true},
				{object: s2, allKeys: true},
			}

			h1, err := calculateConfigHash(c)
			Expect(err).NotTo(HaveOccurred())

			m.Update(cm1, func(obj client.Object) client.Object {
				cm := obj.(*corev1.ConfigMap)
				cm.Data["key1"] = modified

				return cm
			}, timeout).Should(Succeed())
			h2, err := calculateConfigHash(c)
			Expect(err).NotTo(HaveOccurred())

			Expect(h2).NotTo(Equal(h1))
		})

		It("returns a different hash when a single-field child's data is updated", func() {
			c := []configObject{
				{object: cm1, allKeys: false, keys: map[string]struct{}{
					"key1": {},
				},
				},
				{object: cm2, allKeys: true},
				{object: s1, allKeys: false, keys: map[string]struct{}{
					"key1": {},
				},
				},
				{object: s2, allKeys: true},
			}

			h1, err := calculateConfigHash(c)
			Expect(err).NotTo(HaveOccurred())

			m.Update(cm1, func(obj client.Object) client.Object {
				cm := obj.(*corev1.ConfigMap)
				cm.Data["key1"] = modified

				return cm
			}, timeout).Should(Succeed())

			m.Update(s1, func(obj client.Object) client.Object {
				s := obj.(*corev1.Secret)
				s.Data["key1"] = []byte("modified")

				return s
			}, timeout).Should(Succeed())
			h2, err := calculateConfigHash(c)
			Expect(err).NotTo(HaveOccurred())

			Expect(h2).NotTo(Equal(h1))
		})

		It("returns the same hash when a single-field child's data is updated but not for that field", func() {
			c := []configObject{
				{object: cm1, allKeys: false, keys: map[string]struct{}{
					"key1": {},
				},
				},
				{object: cm2, allKeys: true},
				{object: s1, allKeys: false, keys: map[string]struct{}{
					"key1": {},
				},
				},
				{object: s2, allKeys: true},
			}

			h1, err := calculateConfigHash(c)
			Expect(err).NotTo(HaveOccurred())

			m.Update(cm1, func(obj client.Object) client.Object {
				cm := obj.(*corev1.ConfigMap)
				cm.Data["key3"] = modified

				return cm
			}, timeout).Should(Succeed())

			m.Update(s1, func(obj client.Object) client.Object {
				s1 := obj.(*corev1.Secret)
				s1.Data["key3"] = []byte("modified")

				return s1
			}, timeout).Should(Succeed())
			h2, err := calculateConfigHash(c)
			Expect(err).NotTo(HaveOccurred())

			Expect(h2).To(Equal(h1))
		})

		It("returns the same hash when a child's metadata is updated", func() {
			c := []configObject{
				{object: cm1, allKeys: true},
				{object: cm2, allKeys: true},
				{object: s1, allKeys: true},
				{object: s2, allKeys: true},
			}

			h1, err := calculateConfigHash(c)
			Expect(err).NotTo(HaveOccurred())

			m.Update(s1, func(obj client.Object) client.Object {
				s := obj.(*corev1.Secret)
				s.Annotations = map[string]string{"new": "annotations"}

				return s
			}, timeout).Should(Succeed())
			h2, err := calculateConfigHash(c)
			Expect(err).NotTo(HaveOccurred())

			Expect(h2).To(Equal(h1))
		})

		It("returns the same hash independent of child ordering", func() {
			c1 := []configObject{
				{object: cm1, allKeys: true},
				{object: cm2, allKeys: true},
				{object: cm3, allKeys: false, keys: map[string]struct{}{
					"key1": {},
					"key2": {},
				},
				},
				{object: s1, allKeys: true},
				{object: s2, allKeys: true},
				{object: s3, allKeys: false, keys: map[string]struct{}{
					"key1": {},
					"key2": {},
				},
				},
			}
			c2 := []configObject{
				{object: cm1, allKeys: true},
				{object: s2, allKeys: true},
				{object: s3, allKeys: false, keys: map[string]struct{}{
					"key1": {},
					"key2": {},
				},
				},
				{object: cm2, allKeys: true},
				{object: s1, allKeys: true},
				{object: cm3, allKeys: false, keys: map[string]struct{}{
					"key2": {},
					"key1": {},
				},
				},
			}

			h1, err := calculateConfigHash(c1)
			Expect(err).NotTo(HaveOccurred())
			h2, err := calculateConfigHash(c2)
			Expect(err).NotTo(HaveOccurred())

			Expect(h2).To(Equal(h1))
		})
	})

	Context("setConfigHash", func() {
		var deploymentObject *appsv1.Deployment
		var podControllerDeployment podController

		BeforeEach(func() {
			deploymentObject = utils.ExampleDeployment.DeepCopy()
			podControllerDeployment = &deployment{deploymentObject}
		})

		It("sets the hash annotation to the provided value", func() {
			setConfigHash(podControllerDeployment, "1234")

			podAnnotations := deploymentObject.Spec.Template.GetAnnotations()
			Expect(podAnnotations).NotTo(BeNil())

			hash, ok := podAnnotations[ConfigHashAnnotation]
			Expect(ok).To(BeTrue())
			Expect(hash).To(Equal("1234"))
		})

		It("leaves existing annotations in place", func() {
			// Add an annotation to the pod spec
			podAnnotations := deploymentObject.Spec.Template.GetAnnotations()
			if podAnnotations == nil {
				podAnnotations = make(map[string]string)
			}
			podAnnotations["existing"] = "annotation"
			deploymentObject.Spec.Template.SetAnnotations(podAnnotations)

			// Set the config hash
			setConfigHash(podControllerDeployment, "1234")

			// Check the existing annotation is still in place
			podAnnotations = deploymentObject.Spec.Template.GetAnnotations()
			Expect(podAnnotations).NotTo(BeNil())

			hash, ok := podAnnotations["existing"]
			Expect(ok).To(BeTrue())
			Expect(hash).To(Equal("annotation"))
		})
	})
})
