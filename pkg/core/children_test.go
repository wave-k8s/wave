/*
Copyright 2018, 2019 Pusher Ltd.

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
	var deployment *appsv1.Deployment
	var existingChildren []Object
	var currentChildren []ConfigObject
	var mgrStopped *sync.WaitGroup
	var stopMgr chan struct{}

	const timeout = time.Second * 5

	var cm1 *corev1.ConfigMap
	var cm2 *corev1.ConfigMap
	var cm3 *corev1.ConfigMap
	var s1 *corev1.Secret
	var s2 *corev1.Secret
	var s3 *corev1.Secret

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
		s1 = utils.ExampleSecret1.DeepCopy()
		s2 = utils.ExampleSecret2.DeepCopy()
		s3 = utils.ExampleSecret3.DeepCopy()

		m.Create(cm1).Should(Succeed())
		m.Create(cm2).Should(Succeed())
		m.Create(cm3).Should(Succeed())
		m.Create(s1).Should(Succeed())
		m.Create(s2).Should(Succeed())
		m.Create(s3).Should(Succeed())

		deployment = utils.ExampleDeployment.DeepCopy()
		m.Create(deployment).Should(Succeed())

		stopMgr, mgrStopped = StartTestManager(mgr)

		// Ensure the caches have synced
		m.Get(cm1, timeout).Should(Succeed())
		m.Get(cm2, timeout).Should(Succeed())
		m.Get(cm3, timeout).Should(Succeed())
		m.Get(s1, timeout).Should(Succeed())
		m.Get(s2, timeout).Should(Succeed())
		m.Get(s3, timeout).Should(Succeed())
		m.Get(deployment, timeout).Should(Succeed())
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
			currentChildren, err = h.getCurrentChildren(deployment)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns ConfigMaps referenced in Volumes", func() {
			Expect(currentChildren).To(ContainElement(ConfigObject{
				k8sObject:    cm1,
				singleFields: false,
				fieldKeys:    map[string]ConfigField{},
			}))
		})

		It("returns ConfigMaps referenced in EnvFrom", func() {
			Expect(currentChildren).To(ContainElement(ConfigObject{
				k8sObject:    cm2,
				singleFields: false,
				fieldKeys:    map[string]ConfigField{},
			}))
		})

		It("returns ConfigMaps referenced in Env", func() {
			Expect(currentChildren).To(ContainElement(ConfigObject{
				k8sObject:    cm3,
				singleFields: true,
				fieldKeys: map[string]ConfigField{
					"key1": {optional: false},
					"key2": {optional: false},
					"key4": {optional: true},
				},
			}))
		})

		It("returns Secrets referenced in Volumes", func() {
			Expect(currentChildren).To(ContainElement(ConfigObject{
				k8sObject:    s1,
				singleFields: false,
				fieldKeys:    map[string]ConfigField{},
			}))
		})

		It("returns Secrets referenced in EnvFrom", func() {
			Expect(currentChildren).To(ContainElement(ConfigObject{
				k8sObject:    s2,
				singleFields: false,
				fieldKeys:    map[string]ConfigField{},
			}))
		})

		It("returns Secrets referenced in Env", func() {
			Expect(currentChildren).To(ContainElement(ConfigObject{
				k8sObject:    s3,
				singleFields: true,
				fieldKeys: map[string]ConfigField{
					"key1": {optional: false},
					"key2": {optional: false},
					"key4": {optional: true},
				},
			}))
		})

		It("does not return duplicate children", func() {
			Expect(currentChildren).To(HaveLen(6))
		})

		It("returns an error if one of the referenced children is missing", func() {
			// Delete s2 and wait for the cache to sync
			m.Delete(s2).Should(Succeed())
			m.Get(s2, timeout).ShouldNot(Succeed())

			current, err := h.getCurrentChildren(deployment)
			Expect(err).To(HaveOccurred())
			Expect(current).To(BeEmpty())
		})
	})

	Context("getChildNamesByType", func() {
		var configMaps map[string]struct{}
		var secrets map[string]struct{}
		var configMapKeyReferences map[string]map[string]ConfigField
		var secretKeyReferences map[string]map[string]ConfigField

		BeforeEach(func() {
			configMaps, secrets, configMapKeyReferences, secretKeyReferences = getChildNamesByType(deployment)
		})

		It("returns ConfigMaps referenced in Volumes", func() {
			Expect(configMaps).To(HaveKey(cm1.GetName()))
		})

		It("returns ConfigMaps referenced in EnvFrom", func() {
			Expect(configMaps).To(HaveKey(cm2.GetName()))
		})

		It("returns ConfigMaps referenced in Env", func() {
			Expect(configMapKeyReferences).To(HaveKey(cm1.GetName()))
			Expect(configMapKeyReferences[cm1.GetName()]).To(HaveKey("key1"))
			Expect(configMapKeyReferences[cm1.GetName()]["key1"].optional).To(BeFalse())
			Expect(configMapKeyReferences).To(HaveKey(cm3.GetName()))
			Expect(configMapKeyReferences[cm3.GetName()]).To(HaveKey("key1"))
			Expect(configMapKeyReferences[cm3.GetName()]["key1"].optional).To(BeFalse())
			Expect(configMapKeyReferences[cm3.GetName()]).To(HaveKey("key2"))
			Expect(configMapKeyReferences[cm3.GetName()]["key2"].optional).To(BeFalse())
			Expect(configMapKeyReferences[cm3.GetName()]).To(HaveKey("key4"))
			Expect(configMapKeyReferences[cm3.GetName()]["key4"].optional).To(BeTrue())
		})

		It("returns Secrets referenced in Volumes", func() {
			Expect(secrets).To(HaveKey(s1.GetName()))
		})

		It("returns Secrets referenced in EnvFrom", func() {
			Expect(secrets).To(HaveKey(s2.GetName()))
		})

		It("returns Secrets referenced in Env", func() {
			Expect(secretKeyReferences).To(HaveKey(s1.GetName()))
			Expect(secretKeyReferences[s1.GetName()]).To(HaveKey("key1"))
			Expect(configMapKeyReferences[s1.GetName()]["key1"].optional).To(BeFalse())
			Expect(secretKeyReferences).To(HaveKey(s3.GetName()))
			Expect(secretKeyReferences[s3.GetName()]).To(HaveKey("key1"))
			Expect(configMapKeyReferences[s3.GetName()]["key1"].optional).To(BeFalse())
			Expect(secretKeyReferences[s3.GetName()]).To(HaveKey("key2"))
			Expect(configMapKeyReferences[s3.GetName()]["key2"].optional).To(BeFalse())
			Expect(secretKeyReferences[s3.GetName()]).To(HaveKey("key4"))
			Expect(configMapKeyReferences[s3.GetName()]["key4"].optional).To(BeTrue())
		})

		It("does not return extra children", func() {
			Expect(configMaps).To(HaveLen(2))
			Expect(secrets).To(HaveLen(2))
			Expect(configMapKeyReferences).To(HaveLen(2))
			Expect(secretKeyReferences).To(HaveLen(2))
		})
	})

	Context("getExistingChildren", func() {
		BeforeEach(func() {
			m.Get(deployment, timeout).Should(Succeed())
			ownerRef := utils.GetOwnerRef(deployment)

			for _, obj := range []Object{cm1, s1} {
				m.Get(obj, timeout).Should(Succeed())
				obj.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
				m.Update(obj).Should(Succeed())
				m.Eventually(obj, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))
			}

			var err error
			existingChildren, err = h.getExistingChildren(deployment)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns ConfigMaps with the correct OwnerReference", func() {
			Expect(existingChildren).To(ContainElement(cm1))
		})

		It("doesn't return ConfigMaps without OwnerReferences", func() {
			Expect(existingChildren).NotTo(ContainElement(cm2))
			Expect(existingChildren).NotTo(ContainElement(cm3))
		})

		It("returns Secrets with the correct OwnerReference", func() {
			Expect(existingChildren).To(ContainElement(s1))
		})

		It("doesn't return Secrets without OwnerReferences", func() {
			Expect(existingChildren).NotTo(ContainElement(s2))
			Expect(existingChildren).NotTo(ContainElement(s3))
		})

		It("does not return duplicate children", func() {
			Expect(existingChildren).To(HaveLen(2))
		})
	})

	Context("isOwnedBy", func() {
		var ownerRef metav1.OwnerReference
		BeforeEach(func() {
			m.Get(deployment, timeout).Should(Succeed())
			ownerRef = utils.GetOwnerRef(deployment)
		})

		It("returns true when the child has a single owner reference pointing to the owner", func() {
			cm1.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
			Expect(isOwnedBy(cm1, deployment)).To(BeTrue())
		})

		It("returns true when the child has multiple owner references, with one pointing to the owner", func() {
			otherRef := ownerRef
			otherRef.UID = cm1.GetUID()
			cm1.SetOwnerReferences([]metav1.OwnerReference{ownerRef, otherRef})
			Expect(isOwnedBy(cm1, deployment)).To(BeTrue())
		})

		It("returns false when the child has no owner reference pointing to the owner", func() {
			ownerRef.UID = cm1.GetUID()
			cm1.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
			Expect(isOwnedBy(cm1, deployment)).To(BeFalse())
		})
	})

})
