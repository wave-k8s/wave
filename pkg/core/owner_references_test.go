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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/wave-k8s/wave/test/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ = Describe("Wave owner references Suite", func() {
	var c client.Client
	var h *Handler
	var m utils.Matcher
	var deploymentObject *appsv1.Deployment
	var podControllerDeployment podController
	var mgrStopped *sync.WaitGroup
	var stopMgr chan struct{}

	const timeout = time.Second * 5
	const consistentlyTimeout = time.Second

	var cm1 *corev1.ConfigMap
	var cm2 *corev1.ConfigMap
	var cm3 *corev1.ConfigMap
	var s1 *corev1.Secret
	var s2 *corev1.Secret
	var s3 *corev1.Secret
	var ownerRef metav1.OwnerReference

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
		h = NewHandler(c, mgr.GetEventRecorderFor("wave"))
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

		deploymentObject = utils.ExampleDeployment.DeepCopy()
		podControllerDeployment = &deployment{deploymentObject}

		m.Create(deploymentObject).Should(Succeed())

		ownerRef = utils.GetOwnerRefDeployment(deploymentObject)

		stopMgr, mgrStopped = StartTestManager(mgr)

		// Make sure caches have synced
		m.Get(deploymentObject, timeout).Should(Succeed())
	})

	AfterEach(func() {
		close(stopMgr)
		mgrStopped.Wait()

		utils.DeleteAll(cfg, timeout,
			&appsv1.DeploymentList{},
			&corev1.ConfigMapList{},
			&corev1.SecretList{},
			&corev1.EventList{},
		)
	})

	Context("removeOwnerReferences", func() {
		BeforeEach(func() {
			for _, obj := range []Object{cm1, cm2, s1, s2} {
				otherRef := ownerRef.DeepCopy()
				otherRef.UID = obj.GetUID()
				m.Update(obj, func(obj client.Object) client.Object {
					obj.SetOwnerReferences([]metav1.OwnerReference{ownerRef, *otherRef})
					return obj
				}, timeout).Should(Succeed())

				Eventually(obj, timeout).Should(utils.WithOwnerReferences(ConsistOf(ownerRef, *otherRef)))
			}

			children := []Object{cm1, s1}
			err := h.removeOwnerReferences(podControllerDeployment, children)
			Expect(err).NotTo(HaveOccurred())
		})

		It("removes owner references from the list of children given", func() {
			Eventually(cm1, timeout).ShouldNot(utils.WithOwnerReferences(ContainElement(ownerRef)))
			Eventually(s1, timeout).ShouldNot(utils.WithOwnerReferences(ContainElement(ownerRef)))
		})

		It("doesn't remove owner references from children not listed", func() {
			m.Consistently(cm2, consistentlyTimeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))
			m.Consistently(s2, consistentlyTimeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))
		})

		It("doesn't remove owner references pointing to other owners", func() {
			m.Consistently(cm1, consistentlyTimeout).Should(utils.WithOwnerReferences(ContainElement(Not(Equal(ownerRef)))))
			m.Consistently(cm2, consistentlyTimeout).Should(utils.WithOwnerReferences(ContainElement(Not(Equal(ownerRef)))))
			m.Consistently(s1, consistentlyTimeout).Should(utils.WithOwnerReferences(ContainElement(Not(Equal(ownerRef)))))
			m.Consistently(s2, consistentlyTimeout).Should(utils.WithOwnerReferences(ContainElement(Not(Equal(ownerRef)))))
		})

		It("sends events for removing each owner reference", func() {
			cmMessage := "Removing watch for ConfigMap example1"
			sMessage := "Removing watch for Secret example1"
			eventMessage := func(event *corev1.Event) string {
				return event.Message
			}

			Eventually(func() *corev1.EventList {
				events := &corev1.EventList{}
				m.Client.List(context.TODO(), events)
				return events
			}, timeout).Should(utils.WithItems(ContainElement(WithTransform(eventMessage, Equal(cmMessage)))))
			Eventually(func() *corev1.EventList {
				events := &corev1.EventList{}
				m.Client.List(context.TODO(), events)
				return events
			}, timeout).Should(utils.WithItems(ContainElement(WithTransform(eventMessage, Equal(sMessage)))))
		})
	})

	Context("updateOwnerReferences", func() {
		BeforeEach(func() {
			for _, obj := range []Object{cm2, s1, s2} {
				m.Update(obj, func(obj client.Object) client.Object {
					obj.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
					return obj
				}, timeout).Should(Succeed())
				Eventually(obj, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))
			}

			existing := []Object{cm2, cm3, s1, s2}
			current := []configObject{
				{object: cm1, allKeys: true},
				{object: s1, allKeys: true},
				{object: s3, allKeys: false, keys: map[string]struct{}{
					"key1": {},
					"key2": {},
				},
				},
			}
			err := h.updateOwnerReferences(podControllerDeployment, existing, current)
			Expect(err).NotTo(HaveOccurred())
		})

		It("removes owner references from those not in current", func() {
			Eventually(cm2, timeout).ShouldNot(utils.WithOwnerReferences(ContainElement(ownerRef)))
			Eventually(cm3, timeout).ShouldNot(utils.WithOwnerReferences(ContainElement(ownerRef)))
			Eventually(s2, timeout).ShouldNot(utils.WithOwnerReferences(ContainElement(ownerRef)))
		})

		It("adds owner references to those in current", func() {
			Eventually(cm1, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))
			Eventually(s1, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))
			Eventually(s3, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))
		})
	})

	Context("updateOwnerReference", func() {
		BeforeEach(func() {
			// Add an OwnerReference to cm2
			m.Update(cm2, func(obj client.Object) client.Object {
				cm2 := obj.(*corev1.ConfigMap)
				cm2.SetOwnerReferences([]metav1.OwnerReference{ownerRef})

				return cm2
			}, timeout).Should(Succeed())
			Eventually(cm2, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))

			m.Get(cm1, timeout).Should(Succeed())
			m.Get(cm2, timeout).Should(Succeed())
		})

		It("adds an OwnerReference if not present", func() {
			// Add an OwnerReference to cm1
			otherRef := ownerRef
			otherRef.UID = cm1.GetUID()
			m.Update(cm1, func(obj client.Object) client.Object {
				cm1 := obj.(*corev1.ConfigMap)
				cm1.SetOwnerReferences([]metav1.OwnerReference{otherRef})

				return cm1
			}, timeout).Should(Succeed())
			Eventually(cm1, timeout).Should(utils.WithOwnerReferences(ContainElement(otherRef)))

			m.Get(cm1, timeout).Should(Succeed())
			Expect(h.updateOwnerReference(podControllerDeployment, cm1)).NotTo(HaveOccurred())
			Eventually(cm1, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))
		})

		It("doesn't update the child object if there is already and OwnerReference present", func() {
			// Add an OwnerReference to cm2
			m.Update(cm2, func(obj client.Object) client.Object {
				cm2 := obj.(*corev1.ConfigMap)
				cm2.SetOwnerReferences([]metav1.OwnerReference{ownerRef})

				return cm2
			}, timeout).Should(Succeed())
			Eventually(cm2, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))

			// Get the original version
			m.Get(cm2, timeout).Should(Succeed())
			originalVersion := cm2.GetResourceVersion()
			Expect(h.updateOwnerReference(podControllerDeployment, cm2)).NotTo(HaveOccurred())

			// Compare current version
			m.Get(cm2, timeout).Should(Succeed())
			Expect(cm2.GetResourceVersion()).To(Equal(originalVersion))
		})

		It("sends events for adding each owner reference", func() {
			m.Get(cm1, timeout).Should(Succeed())
			Expect(h.updateOwnerReference(podControllerDeployment, cm1)).NotTo(HaveOccurred())
			Eventually(cm1, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))

			cmMessage := "Adding watch for ConfigMap example1"
			eventMessage := func(event *corev1.Event) string {
				return event.Message
			}

			Eventually(func() *corev1.EventList {
				events := &corev1.EventList{}
				m.Client.List(context.TODO(), events)
				return events
			}, timeout).Should(utils.WithItems(ContainElement(WithTransform(eventMessage, Equal(cmMessage)))))
		})
	})

	Context("getOrphans", func() {
		It("returns an empty list when current and existing match", func() {
			current := []configObject{
				{object: cm1, allKeys: true},
				{object: cm2, allKeys: true},
				{object: s1, allKeys: true},
				{object: s2, allKeys: true},
			}
			existing := []Object{cm1, cm2, s1, s2}
			Expect(getOrphans(existing, current)).To(BeEmpty())
		})

		It("returns an empty list when existing is a subset of current", func() {
			current := []configObject{
				{object: cm1, allKeys: true},
				{object: cm2, allKeys: true},
				{object: s1, allKeys: true},
				{object: s2, allKeys: true},
			}
			existing := []Object{cm1, s2}
			Expect(getOrphans(existing, current)).To(BeEmpty())
		})

		It("returns the correct objects when current is a subset of existing", func() {
			current := []configObject{
				{object: cm1, allKeys: true},
				{object: s2, allKeys: true},
			}
			existing := []Object{cm1, cm2, s1, s2}
			orphans := getOrphans(existing, current)
			Expect(orphans).To(ContainElement(cm2))
			Expect(orphans).To(ContainElement(s1))
		})

		Context("when current contains multiple singleField entries", func() {
			It("returns an empty list when current and existing match", func() {
				current := []configObject{
					{object: cm1, allKeys: true},
					{object: cm2, allKeys: false, keys: map[string]struct{}{
						"key1": {},
						"key2": {},
					},
					},
					{object: s1, allKeys: true},
					{object: s2, allKeys: true},
				}
				existing := []Object{cm1, cm2, s1, s2}
				Expect(getOrphans(existing, current)).To(BeEmpty())
			})

			It("returns an empty list when existing is a subset of current", func() {
				current := []configObject{
					{object: cm1, allKeys: true},
					{object: cm2, allKeys: false, keys: map[string]struct{}{
						"key1": {},
						"key2": {},
					},
					},
					{object: s1, allKeys: true},
					{object: s2, allKeys: true},
				}
				existing := []Object{cm1, s2}
				Expect(getOrphans(existing, current)).To(BeEmpty())
			})

			It("returns the correct objects when current is a subset of existing", func() {
				current := []configObject{
					{object: cm1, allKeys: true},
					{object: s2, allKeys: false, keys: map[string]struct{}{
						"key1": {},
						"key2": {},
					},
					},
				}
				existing := []Object{cm1, cm2, s1, s2}
				orphans := getOrphans(existing, current)
				Expect(orphans).To(ContainElement(cm2))
				Expect(orphans).To(ContainElement(s1))
			})
		})
	})

	Context("getOwnerReference", func() {
		var ref metav1.OwnerReference
		BeforeEach(func() {
			ref = getOwnerReference(podControllerDeployment)
		})

		It("sets the APIVersion", func() {
			Expect(ref.APIVersion).To(Equal("apps/v1"))
		})

		It("sets the Kind", func() {
			Expect(ref.Kind).To(Equal("Deployment"))
		})

		It("sets the UID", func() {
			Expect(ref.UID).To(Equal(deploymentObject.UID))
		})

		It("sets the Name", func() {
			Expect(ref.Name).To(Equal(deploymentObject.Name))
		})

		It("sets Controller to false", func() {
			Expect(ref.Controller).NotTo(BeNil())
			Expect(*ref.Controller).To(BeFalse())
		})

		It("sets BlockOwnerDeletion to true", func() {
			Expect(ref.BlockOwnerDeletion).NotTo(BeNil())
			Expect(*ref.BlockOwnerDeletion).To(BeTrue())
		})
	})
})
