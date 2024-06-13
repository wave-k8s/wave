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
	"context"
	"sync"
	"time"

	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wave-k8s/wave/test/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ = Describe("Wave children Suite", func() {
	var c client.Client
	var h *Handler[*appsv1.Deployment]
	var m utils.Matcher
	var deploymentObject *appsv1.Deployment
	var mgrStopped *sync.WaitGroup
	var stopMgr chan struct{}

	const timeout = time.Second * 5

	var cm1 *corev1.ConfigMap
	var cm2 *corev1.ConfigMap
	var cm3 *corev1.ConfigMap
	var cm4 *corev1.ConfigMap
	var cm5 *corev1.ConfigMap
	var cm6 *corev1.ConfigMap
	var s1 *corev1.Secret
	var s2 *corev1.Secret
	var s3 *corev1.Secret
	var s4 *corev1.Secret
	var s5 *corev1.Secret
	var s6 *corev1.Secret

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
		c = mgr.GetClient()
		//		h = NewHandler(c, mgr.GetEventRecorderFor("wave"))
		h = NewHandler[*appsv1.Deployment](mgr.GetClient(), mgr.GetEventRecorderFor("wave"))

		m = utils.Matcher{Client: c}

		// Create some configmaps and secrets
		cm1 = utils.ExampleConfigMap1.DeepCopy()
		cm2 = utils.ExampleConfigMap2.DeepCopy()
		cm3 = utils.ExampleConfigMap3.DeepCopy()
		cm4 = utils.ExampleConfigMap4.DeepCopy()
		cm5 = utils.ExampleConfigMap5.DeepCopy()
		cm6 = utils.ExampleConfigMap6.DeepCopy()
		s1 = utils.ExampleSecret1.DeepCopy()
		s2 = utils.ExampleSecret2.DeepCopy()
		s3 = utils.ExampleSecret3.DeepCopy()
		s4 = utils.ExampleSecret4.DeepCopy()
		s5 = utils.ExampleSecret5.DeepCopy()
		s6 = utils.ExampleSecret6.DeepCopy()

		m.Create(cm1).Should(Succeed())
		m.Create(cm2).Should(Succeed())
		m.Create(cm3).Should(Succeed())
		m.Create(cm4).Should(Succeed())
		m.Create(cm5).Should(Succeed())
		m.Create(cm6).Should(Succeed())
		m.Create(s1).Should(Succeed())
		m.Create(s2).Should(Succeed())
		m.Create(s3).Should(Succeed())
		m.Create(s4).Should(Succeed())
		m.Create(s5).Should(Succeed())
		m.Create(s6).Should(Succeed())

		deploymentObject = utils.ExampleDeployment.DeepCopy()

		m.Create(deploymentObject).Should(Succeed())

		stopMgr, mgrStopped = StartTestManager(mgr)

		// Ensure the caches have synced
		m.Get(cm1, timeout).Should(Succeed())
		m.Get(cm2, timeout).Should(Succeed())
		m.Get(cm3, timeout).Should(Succeed())
		m.Get(cm4, timeout).Should(Succeed())
		m.Get(cm5, timeout).Should(Succeed())
		m.Get(cm6, timeout).Should(Succeed())
		m.Get(s1, timeout).Should(Succeed())
		m.Get(s2, timeout).Should(Succeed())
		m.Get(s3, timeout).Should(Succeed())
		m.Get(s4, timeout).Should(Succeed())
		m.Get(s5, timeout).Should(Succeed())
		m.Get(s6, timeout).Should(Succeed())
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

	Context("getChildNamesByType and getCurrentChildren", func() {
		var configMaps map[types.NamespacedName]*corev1.ConfigMap
		var secrets map[types.NamespacedName]*corev1.Secret
		var configMapsConfig configMetadataList
		var secretsConfig configMetadataList

		BeforeEach(func() {
			var err error
			configMapsConfig, secretsConfig = getChildNamesByType(deploymentObject)
			configMaps, secrets, err = h.getCurrentChildren(configMapsConfig, secretsConfig)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate", func() {
			Expect(h.checkRequiredChildren(configMaps, secrets, configMapsConfig, secretsConfig)).To(Succeed())
		})

		It("returns ConfigMaps referenced in Volumes", func() {
			Expect(configMaps[GetNamespacedNameFromObject(cm1)]).To(Equal(cm1))
			Expect(configMapsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedNameFromObject(cm1),
				required: true,
				allKeys:  true,
			}))
		})

		It("returns ConfigMaps referenced in Volume Projections", func() {
			Expect(configMaps[GetNamespacedNameFromObject(cm6)]).To(Equal(cm6))
			Expect(configMapsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedNameFromObject(cm6),
				required: true,
				allKeys:  false,
				keys: map[string]struct{}{
					"key1": {},
					"key3": {},
				},
			}))
			Expect(configMaps[GetNamespacedNameFromObject(cm5)]).To(Equal(cm5))
			Expect(configMapsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedNameFromObject(cm5),
				required: false,
				allKeys:  true,
			}))
		})

		It("returns ConfigMaps referenced in EnvFrom", func() {
			Expect(configMaps[GetNamespacedNameFromObject(cm2)]).To(Equal(cm2))
			Expect(configMapsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedNameFromObject(cm2),
				required: true,
				allKeys:  true,
			}))
		})

		It("returns ConfigMaps referenced in Env", func() {
			Expect(configMaps[GetNamespacedNameFromObject(cm3)]).To(Equal(cm3))
			Expect(configMapsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedNameFromObject(cm3),
				required: true,
				allKeys:  false,
				keys: map[string]struct{}{
					"key1": {},
				},
			}))
			Expect(configMapsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedNameFromObject(cm3),
				required: true,
				allKeys:  false,
				keys: map[string]struct{}{
					"key2": {},
				},
			}))
			Expect(configMapsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedNameFromObject(cm3),
				required: false,
				allKeys:  false,
				keys: map[string]struct{}{
					"key4": {},
				},
			}))
			Expect(configMaps[GetNamespacedNameFromObject(cm4)]).To(Equal(cm4))
			Expect(configMapsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedNameFromObject(cm4),
				required: false,
				allKeys:  false,
				keys: map[string]struct{}{
					"key1": {},
				},
			}))
		})

		It("returns Secrets referenced in Volumes", func() {
			Expect(secrets[GetNamespacedNameFromObject(s1)]).To(Equal(s1))
			Expect(secretsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedNameFromObject(s1),
				required: true,
				allKeys:  true,
			}))
		})

		It("returns Secrets referenced in Volume Projections", func() {
			Expect(secrets[GetNamespacedNameFromObject(s6)]).To(Equal(s6))
			Expect(secretsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedNameFromObject(s6),
				required: true,
				allKeys:  false,
				keys: map[string]struct{}{
					"key1": {},
					"key3": {},
				},
			}))
			Expect(secrets[GetNamespacedNameFromObject(s5)]).To(Equal(s5))
			Expect(secretsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedNameFromObject(s5),
				required: false,
				allKeys:  true,
			}))
		})

		It("returns Secrets referenced in EnvFrom", func() {
			Expect(secrets[GetNamespacedNameFromObject(s2)]).To(Equal(s2))
			Expect(secretsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedNameFromObject(s2),
				required: true,
				allKeys:  true,
			}))
		})

		It("returns Secrets referenced in Env", func() {
			Expect(secrets[GetNamespacedNameFromObject(s3)]).To(Equal(s3))
			Expect(secretsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedNameFromObject(s3),
				required: true,
				allKeys:  false,
				keys: map[string]struct{}{
					"key1": {},
				},
			}))
			Expect(secretsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedNameFromObject(s3),
				required: true,
				allKeys:  false,
				keys: map[string]struct{}{
					"key2": {},
				},
			}))
			Expect(secretsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedNameFromObject(s3),
				required: false,
				allKeys:  false,
				keys: map[string]struct{}{
					"key4": {},
				},
			}))
			Expect(secrets[GetNamespacedNameFromObject(s4)]).To(Equal(s4))
			Expect(secretsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedNameFromObject(s4),
				required: false,
				allKeys:  false,
				keys: map[string]struct{}{
					"key1": {},
				},
			}))
		})

		It("returns ConfigMaps referenced in extra-configmaps annotations", func() {
			Expect(configMaps[GetNamespacedName("test-cm1", "ns1")]).To(BeNil())
			Expect(configMapsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedName("test-cm1", "ns1"),
				required: false,
				allKeys:  true,
			}))
			Expect(configMaps[GetNamespacedName("test-cm2", "ns2")]).To(BeNil())
			Expect(configMapsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedName("test-cm2", "ns2"),
				required: false,
				allKeys:  true,
			}))
			Expect(configMaps[GetNamespacedName("local-cm1", deploymentObject.GetNamespace())]).To(BeNil())
			Expect(configMapsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedName("local-cm1", deploymentObject.GetNamespace()),
				required: false,
				allKeys:  true,
			}))
		})

		It("returns Secrets referenced in extra-secrets annotations", func() {
			Expect(secrets[GetNamespacedName("test-secret1", "ns1")]).To(BeNil())
			Expect(secretsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedName("test-secret1", "ns1"),
				required: false,
				allKeys:  true,
			}))
			Expect(secrets[GetNamespacedName("test-secret2", "ns2")]).To(BeNil())
			Expect(secretsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedName("test-secret2", "ns2"),
				required: false,
				allKeys:  true,
			}))
			Expect(secrets[GetNamespacedName("local-secret1", deploymentObject.GetNamespace())]).To(BeNil())
			Expect(secretsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedName("local-secret1", deploymentObject.GetNamespace()),
				required: false,
				allKeys:  true,
			}))
		})

		It("optional ConfigMaps referenced in Volumes are returned as optional", func() {
			Expect(configMaps[GetNamespacedName("volume-optional", deploymentObject.GetNamespace())]).To(BeNil())
			Expect(configMapsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedName("volume-optional", deploymentObject.GetNamespace()),
				required: false,
				allKeys:  true,
			}))
		})

		It("optional Secrets referenced in Volumes are returned as optional", func() {
			Expect(secrets[GetNamespacedName("volume-optional", deploymentObject.GetNamespace())]).To(BeNil())
			Expect(secretsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedName("volume-optional", deploymentObject.GetNamespace()),
				required: false,
				allKeys:  true,
			}))
		})

		It("optional ConfigMaps referenced in EnvFrom are returned as optional", func() {
			Expect(configMaps[GetNamespacedName("envfrom-optional", deploymentObject.GetNamespace())]).To(BeNil())
			Expect(configMapsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedName("envfrom-optional", deploymentObject.GetNamespace()),
				required: false,
				allKeys:  true,
			}))
		})

		It("returns ConfigMaps referenced in Env as optional correctly", func() {
			Expect(configMaps[GetNamespacedName("env-optional", deploymentObject.GetNamespace())]).To(BeNil())
			Expect(configMapsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedName("env-optional", deploymentObject.GetNamespace()),
				required: false,
				allKeys:  false,
				keys: map[string]struct{}{
					"key2": {},
				},
			}))
		})

		It("optional Secrets referenced in EnvFrom are returned as optional", func() {
			Expect(secrets[GetNamespacedName("envfrom-optional", deploymentObject.GetNamespace())]).To(BeNil())
			Expect(secretsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedName("envfrom-optional", deploymentObject.GetNamespace()),
				required: false,
				allKeys:  true,
			}))
		})

		It("returns secrets referenced in Env as optional correctly", func() {
			Expect(secrets[GetNamespacedName("env-optional", deploymentObject.GetNamespace())]).To(BeNil())
			Expect(secretsConfig).To(ContainElement(configMetadata{
				name:     GetNamespacedName("env-optional", deploymentObject.GetNamespace()),
				required: false,
				allKeys:  false,
				keys: map[string]struct{}{
					"key2": {},
				},
			}))
		})

		It("does not return duplicate children", func() {
			Expect(secrets).To(HaveLen(6))
			Expect(configMaps).To(HaveLen(6))
		})

		It("should fail validation if a children is missing", func() {
			// Delete s2 and wait for the cache to sync
			m.Delete(s2).Should(Succeed())
			m.Get(s2, timeout).ShouldNot(Succeed())

			configMapsConfig, secretsConfig = getChildNamesByType(deploymentObject)
			configMaps, secrets, err := h.getCurrentChildren(configMapsConfig, secretsConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(secrets).To(HaveLen(5))
			Expect(configMaps).To(HaveLen(6))
			Expect(h.checkRequiredChildren(configMaps, secrets, configMapsConfig, secretsConfig)).To(MatchError("not all required children exist: missing required secret default/example2"))
		})

		It("should fail validation if a children is missing a required key", func() {
			// Delete s2 and wait for the cache to sync
			cm6WithoutKey3 := utils.ExampleConfigMap6WithoutKey3.DeepCopy()

			m.Get(cm6, timeout).Should(Succeed())
			cm6.Data = cm6WithoutKey3.Data
			Expect(m.Client.Update(context.TODO(), cm6)).Should(Succeed())

			configMapsConfig, secretsConfig = getChildNamesByType(deploymentObject)
			configMaps, secrets, err := h.getCurrentChildren(configMapsConfig, secretsConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(secrets).To(HaveLen(6))
			Expect(configMaps).To(HaveLen(6))
			Expect(h.checkRequiredChildren(configMaps, secrets, configMapsConfig, secretsConfig)).To(MatchError("not all required children exist: missing required key key3 in configmap default/example6"))
		})
	})

	Context("getExistingChildren (deprecated)", func() {
		BeforeEach(func() {
			m.Get(deploymentObject, timeout).Should(Succeed())
			ownerRef := utils.GetOwnerRefDeployment(deploymentObject)

			for _, obj := range []Object{cm1, s1} {
				m.Update(obj, func(obj client.Object) client.Object {
					obj.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
					return obj
				}, timeout).Should(Succeed())
				Eventually(obj, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))
			}

			var err error
			_, err = h.getExistingChildren(deploymentObject)
			Expect(err).NotTo(HaveOccurred())
		})

		existingChildrenFn := func() ([]Object, error) {
			return h.getExistingChildren(deploymentObject)
		}

		It("returns ConfigMaps with the correct OwnerReference", func() {
			Eventually(existingChildrenFn).Should(ContainElement(cm1))
		})

		It("doesn't return ConfigMaps without OwnerReferences", func() {
			Eventually(existingChildrenFn).ShouldNot(ContainElement(cm2))
			Eventually(existingChildrenFn).ShouldNot(ContainElement(cm3))
			Eventually(existingChildrenFn).ShouldNot(ContainElement(cm4))
		})

		It("returns Secrets with the correct OwnerReference", func() {
			Eventually(existingChildrenFn).Should(ContainElement(s1))
		})

		It("doesn't return Secrets without OwnerReferences", func() {
			Eventually(existingChildrenFn).ShouldNot(ContainElement(s2))
			Eventually(existingChildrenFn).ShouldNot(ContainElement(s3))
			Eventually(existingChildrenFn).ShouldNot(ContainElement(s4))
		})

		It("does not return duplicate children", func() {
			Eventually(existingChildrenFn).Should(HaveLen(2))
		})
	})

	Context("isOwnedBy (deprecated)", func() {
		var ownerRef metav1.OwnerReference
		BeforeEach(func() {
			m.Get(deploymentObject, timeout).Should(Succeed())
			ownerRef = utils.GetOwnerRefDeployment(deploymentObject)
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
