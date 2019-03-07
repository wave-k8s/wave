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

package core

import (
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pusher/wave/test/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ = Describe("Wave children Suite", func() {
	var c client.Client
	var h *Handler
	var m utils.Matcher
	var deploymentObject *appsv1.Deployment
	var podControllerDeployment podController
	var existingChildren []Object
	var currentChildren []configObject
	var mgrStopped *sync.WaitGroup
	var stopMgr chan struct{}

	const timeout = time.Second * 5

	var cm1 *corev1.ConfigMap
	var cm2 *corev1.ConfigMap
	var cm3 *corev1.ConfigMap
	var cm4 *corev1.ConfigMap
	var s1 *corev1.Secret
	var s2 *corev1.Secret
	var s3 *corev1.Secret
	var s4 *corev1.Secret

	BeforeEach(func() {
		mgr, err := manager.New(cfg, manager.Options{})
		Expect(err).NotTo(HaveOccurred())
		c = mgr.GetClient()
		h = NewHandler(c, mgr.GetRecorder("wave"))
		m = utils.Matcher{Client: c}

		// Create some configmaps and secrets
		cm1 = utils.ExampleConfigMap1.DeepCopy()
		cm2 = utils.ExampleConfigMap2.DeepCopy()
		cm3 = utils.ExampleConfigMap3.DeepCopy()
		cm4 = utils.ExampleConfigMap4.DeepCopy()
		s1 = utils.ExampleSecret1.DeepCopy()
		s2 = utils.ExampleSecret2.DeepCopy()
		s3 = utils.ExampleSecret3.DeepCopy()
		s4 = utils.ExampleSecret4.DeepCopy()

		m.Create(cm1).Should(Succeed())
		m.Create(cm2).Should(Succeed())
		m.Create(cm3).Should(Succeed())
		m.Create(cm4).Should(Succeed())
		m.Create(s1).Should(Succeed())
		m.Create(s2).Should(Succeed())
		m.Create(s3).Should(Succeed())
		m.Create(s4).Should(Succeed())

		deploymentObject = utils.ExampleDeployment.DeepCopy()
		podControllerDeployment = &deployment{deploymentObject}

		m.Create(deploymentObject).Should(Succeed())

		stopMgr, mgrStopped = StartTestManager(mgr)

		// Ensure the caches have synced
		m.Get(cm1, timeout).Should(Succeed())
		m.Get(cm2, timeout).Should(Succeed())
		m.Get(cm3, timeout).Should(Succeed())
		m.Get(cm4, timeout).Should(Succeed())
		m.Get(s1, timeout).Should(Succeed())
		m.Get(s2, timeout).Should(Succeed())
		m.Get(s3, timeout).Should(Succeed())
		m.Get(s4, timeout).Should(Succeed())
		m.Get(deploymentObject, timeout).Should(Succeed())
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

	Context("getCurrentChildren", func() {
		BeforeEach(func() {
			var err error
			currentChildren, err = h.getCurrentChildren(podControllerDeployment)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns ConfigMaps referenced in Volumes", func() {
			Expect(currentChildren).To(ContainElement(configObject{
				object:   cm1,
				required: true,
				allKeys:  true,
			}))
		})

		It("returns ConfigMaps referenced in EnvFrom", func() {
			Expect(currentChildren).To(ContainElement(configObject{
				object:   cm2,
				required: true,
				allKeys:  true,
			}))
		})

		It("returns ConfigMaps referenced in Env", func() {
			Expect(currentChildren).To(ContainElement(configObject{
				object:   cm3,
				required: true,
				allKeys:  false,
				keys: map[string]struct{}{
					"key1": {},
					"key2": {},
					"key4": {},
				},
			}))
			Expect(currentChildren).To(ContainElement(configObject{
				object:   cm4,
				required: false,
				allKeys:  false,
				keys: map[string]struct{}{
					"key1": {},
				},
			}))
		})

		It("returns Secrets referenced in Volumes", func() {
			Expect(currentChildren).To(ContainElement(configObject{
				object:   s1,
				required: true,
				allKeys:  true,
			}))
		})

		It("returns Secrets referenced in EnvFrom", func() {
			Expect(currentChildren).To(ContainElement(configObject{
				object:   s2,
				required: true,
				allKeys:  true,
			}))
		})

		It("returns Secrets referenced in Env", func() {
			Expect(currentChildren).To(ContainElement(configObject{
				object:   s3,
				required: true,
				allKeys:  false,
				keys: map[string]struct{}{
					"key1": {},
					"key2": {},
					"key4": {},
				},
			}))
			Expect(currentChildren).To(ContainElement(configObject{
				object:   s4,
				required: false,
				allKeys:  false,
				keys: map[string]struct{}{
					"key1": {},
				},
			}))
		})

		It("does not return duplicate children", func() {
			Expect(currentChildren).To(HaveLen(8))
		})

		It("returns an error if one of the referenced children is missing", func() {
			// Delete s2 and wait for the cache to sync
			m.Delete(s2).Should(Succeed())
			m.Get(s2, timeout).ShouldNot(Succeed())

			current, err := h.getCurrentChildren(podControllerDeployment)
			Expect(err).To(HaveOccurred())
			Expect(current).To(BeEmpty())
		})
	})

	Context("getChildNamesByType", func() {
		var configMaps map[string]configMetadata
		var secrets map[string]configMetadata

		BeforeEach(func() {
			configMaps, secrets = getChildNamesByType(podControllerDeployment)
		})

		It("returns ConfigMaps referenced in Volumes", func() {
			Expect(configMaps).To(HaveKeyWithValue(cm1.GetName(), configMetadata{required: true, allKeys: true}))
		})

		It("returns ConfigMaps referenced in EnvFrom", func() {
			Expect(configMaps).To(HaveKeyWithValue(cm2.GetName(), configMetadata{required: true, allKeys: true}))
		})

		It("returns ConfigMaps referenced in Env", func() {
			Expect(configMaps).To(HaveKeyWithValue(cm3.GetName(), configMetadata{
				required: true,
				allKeys:  false,
				keys: map[string]struct{}{
					"key1": {},
					"key2": {},
					"key4": {},
				},
			}))
		})

		It("returns Secrets referenced in Volumes", func() {
			Expect(secrets).To(HaveKeyWithValue(s1.GetName(), configMetadata{required: true, allKeys: true}))
		})

		It("returns Secrets referenced in EnvFrom", func() {
			Expect(secrets).To(HaveKeyWithValue(s2.GetName(), configMetadata{required: true, allKeys: true}))
		})

		It("returns Secrets referenced in Env", func() {
			Expect(configMaps).To(HaveKeyWithValue(s3.GetName(), configMetadata{
				required: true,
				allKeys:  false,
				keys: map[string]struct{}{
					"key1": {},
					"key2": {},
					"key4": {},
				},
			}))
		})

		It("does not return extra children", func() {
			Expect(configMaps).To(HaveLen(4))
			Expect(secrets).To(HaveLen(4))
		})
	})

	Context("getExistingChildren", func() {
		BeforeEach(func() {
			m.Get(deploymentObject, timeout).Should(Succeed())
			ownerRef := utils.GetOwnerRef(deploymentObject)

			for _, obj := range []Object{cm1, s1} {
				m.Get(obj, timeout).Should(Succeed())
				obj.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
				m.Update(obj).Should(Succeed())
				m.Eventually(obj, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))
			}

			var err error
			existingChildren, err = h.getExistingChildren(podControllerDeployment)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns ConfigMaps with the correct OwnerReference", func() {
			Expect(existingChildren).To(ContainElement(cm1))
		})

		It("doesn't return ConfigMaps without OwnerReferences", func() {
			Expect(existingChildren).NotTo(ContainElement(cm2))
			Expect(existingChildren).NotTo(ContainElement(cm3))
			Expect(existingChildren).NotTo(ContainElement(cm4))
		})

		It("returns Secrets with the correct OwnerReference", func() {
			Expect(existingChildren).To(ContainElement(s1))
		})

		It("doesn't return Secrets without OwnerReferences", func() {
			Expect(existingChildren).NotTo(ContainElement(s2))
			Expect(existingChildren).NotTo(ContainElement(s3))
			Expect(existingChildren).NotTo(ContainElement(s4))
		})

		It("does not return duplicate children", func() {
			Expect(existingChildren).To(HaveLen(2))
		})
	})

	Context("isOwnedBy", func() {
		var ownerRef metav1.OwnerReference
		BeforeEach(func() {
			m.Get(deploymentObject, timeout).Should(Succeed())
			ownerRef = utils.GetOwnerRef(deploymentObject)
		})

		It("returns true when the child has a single owner reference pointing to the owner", func() {
			cm1.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
			Expect(isOwnedBy(cm1, deploymentObject)).To(BeTrue())
		})

		It("returns true when the child has multiple owner references, with one pointing to the owner", func() {
			otherRef := ownerRef
			otherRef.UID = cm1.GetUID()
			cm1.SetOwnerReferences([]metav1.OwnerReference{ownerRef, otherRef})
			Expect(isOwnedBy(cm1, deploymentObject)).To(BeTrue())
		})

		It("returns false when the child has no owner reference pointing to the owner", func() {
			ownerRef.UID = cm1.GetUID()
			cm1.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
			Expect(isOwnedBy(cm1, deploymentObject)).To(BeFalse())
		})
	})

})
