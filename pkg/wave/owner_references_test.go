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

package wave

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

var _ = Describe("Wave owner references Suite", func() {
	var c client.Client
	var m utils.Matcher
	var deployment *appsv1.Deployment
	var mgrStopped *sync.WaitGroup
	var stopMgr chan struct{}

	const timeout = time.Second * 5
	const consistentlyTimeout = time.Second

	var cm1 *corev1.ConfigMap
	var cm2 *corev1.ConfigMap
	var s1 *corev1.Secret
	var s2 *corev1.Secret
	var ownerRef metav1.OwnerReference

	BeforeEach(func() {
		mgr, err := manager.New(cfg, manager.Options{})
		Expect(err).NotTo(HaveOccurred())
		c = mgr.GetClient()
		m = utils.Matcher{Client: c}

		reconciler := newReconciler(mgr)
		Expect(add(mgr, reconciler)).NotTo(HaveOccurred())

		var ok bool
		r, ok = reconciler.(*ReconcileDeployment)
		Expect(ok).To(BeTrue())

		// Create some configmaps and secrets
		cm1 = utils.ExampleConfigMap1.DeepCopy()
		cm2 = utils.ExampleConfigMap2.DeepCopy()
		s1 = utils.ExampleSecret1.DeepCopy()
		s2 = utils.ExampleSecret2.DeepCopy()

		m.Create(cm1).Should(Succeed())
		m.Create(cm2).Should(Succeed())
		m.Create(s1).Should(Succeed())
		m.Create(s2).Should(Succeed())

		deployment = utils.ExampleDeployment.DeepCopy()
		m.Create(deployment).Should(Succeed())

		ownerRef = utils.GetOwnerRef(deployment)

		stopMgr, mgrStopped = StartTestManager(mgr)

		// Make sure caches have synced
		m.Get(deployment, timeout).Should(Succeed())
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
			for _, obj := range []object{cm1, cm2, s1, s2} {
				otherRef := ownerRef.DeepCopy()
				otherRef.UID = obj.GetUID()
				obj.SetOwnerReferences([]metav1.OwnerReference{ownerRef, *otherRef})
				m.Update(obj).Should(Succeed())

				m.Eventually(obj, timeout).Should(utils.WithOwnerReferences(ConsistOf(ownerRef, *otherRef)))
			}

			children := []object{cm1, s1}
			err := r.removeOwnerReferences(deployment, children)
			Expect(err).NotTo(HaveOccurred())
		})

		It("removes owner references from the list of children given", func() {
			m.Eventually(cm1, timeout).ShouldNot(utils.WithOwnerReferences(ContainElement(ownerRef)))
			m.Eventually(s1, timeout).ShouldNot(utils.WithOwnerReferences(ContainElement(ownerRef)))
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
			events := &corev1.EventList{}
			cmMessage := "Removing watch for ConfigMap example1"
			sMessage := "Removing watch for Secret example1"
			eventMessage := func(event *corev1.Event) string {
				return event.Message
			}

			m.Eventually(events, timeout).Should(utils.WithItems(ContainElement(WithTransform(eventMessage, Equal(cmMessage)))))
			m.Eventually(events, timeout).Should(utils.WithItems(ContainElement(WithTransform(eventMessage, Equal(sMessage)))))
		})
	})

	Context("updateOwnerReferences", func() {
		BeforeEach(func() {
			for _, obj := range []object{cm2, s1, s2} {
				obj.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
				m.Update(obj).Should(Succeed())

				m.Eventually(obj, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))
			}

			existing := []object{cm2, s1, s2}
			current := []object{cm1, s1}
			err := r.updateOwnerReferences(deployment, existing, current)
			Expect(err).NotTo(HaveOccurred())
		})

		It("removes owner references from those not in current", func() {
			m.Eventually(cm2, timeout).ShouldNot(utils.WithOwnerReferences(ContainElement(ownerRef)))
			m.Eventually(s2, timeout).ShouldNot(utils.WithOwnerReferences(ContainElement(ownerRef)))
		})

		It("adds owner references to those in current", func() {
			m.Eventually(cm1, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))
			m.Eventually(s1, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))
		})
	})

	Context("updateOwnerReference", func() {
		BeforeEach(func() {
			// Add an OwnerReference to cm2
			cm2.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
			m.Update(cm2).Should(Succeed())
			m.Eventually(cm2, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))

			m.Get(cm1, timeout).Should(Succeed())
			m.Get(cm2, timeout).Should(Succeed())
		})

		It("adds an OwnerReference if not present", func() {
			// Add an OwnerReference to cm1
			otherRef := ownerRef
			otherRef.UID = cm1.GetUID()
			cm1.SetOwnerReferences([]metav1.OwnerReference{otherRef})
			m.Update(cm1).Should(Succeed())
			m.Eventually(cm1, timeout).Should(utils.WithOwnerReferences(ContainElement(otherRef)))

			m.Get(cm1, timeout).Should(Succeed())
			Expect(r.updateOwnerReference(deployment, cm1)).NotTo(HaveOccurred())
			m.Eventually(cm1, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))
		})

		It("doesn't update the child object if there is already and OwnerReference present", func() {
			// Add an OwnerReference to cm2
			cm2.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
			m.Update(cm2).Should(Succeed())
			m.Eventually(cm2, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))

			// Get the original version
			m.Get(cm2, timeout).Should(Succeed())
			originalVersion := cm2.GetResourceVersion()
			Expect(r.updateOwnerReference(deployment, cm2)).NotTo(HaveOccurred())

			// Compare current version
			m.Get(cm2, timeout).Should(Succeed())
			Expect(cm2.GetResourceVersion()).To(Equal(originalVersion))
		})

		It("sends events for adding each owner reference", func() {
			m.Get(cm1, timeout).Should(Succeed())
			Expect(r.updateOwnerReference(deployment, cm1)).NotTo(HaveOccurred())
			m.Eventually(cm1, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))

			events := &corev1.EventList{}
			cmMessage := "Adding watch for ConfigMap example1"
			eventMessage := func(event *corev1.Event) string {
				return event.Message
			}

			m.Eventually(events, timeout).Should(utils.WithItems(ContainElement(WithTransform(eventMessage, Equal(cmMessage)))))
		})
	})

	Context("getOrphans", func() {
		It("returns an empty list when current and existing match", func() {
			current := []object{cm1, cm2, s1, s2}
			existing := current
			Expect(getOrphans(existing, current)).To(BeEmpty())
		})

		It("returns an empty list when existing is a subset of current", func() {
			existing := []object{cm1, s2}
			current := append(existing, cm2, s1)
			Expect(getOrphans(existing, current)).To(BeEmpty())
		})

		It("returns the correct objects when current is a subset of existing", func() {
			current := []object{cm1, s2}
			existing := append(current, cm2, s1)
			orphans := getOrphans(existing, current)
			Expect(orphans).To(ContainElement(cm2))
			Expect(orphans).To(ContainElement(s1))
		})
	})

	Context("getOwnerReference", func() {
		var ref metav1.OwnerReference
		BeforeEach(func() {
			ref = getOwnerReference(deployment)
		})

		It("sets the APIVersion", func() {
			Expect(ref.APIVersion).To(Equal("apps/v1"))
		})

		It("sets the Kind", func() {
			Expect(ref.Kind).To(Equal("Deployment"))
		})

		It("sets the UID", func() {
			Expect(ref.UID).To(Equal(deployment.UID))
		})

		It("sets the Name", func() {
			Expect(ref.Name).To(Equal(deployment.Name))
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
