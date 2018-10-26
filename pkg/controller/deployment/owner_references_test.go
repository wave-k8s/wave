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

package deployment

import (
	"context"
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pusher/wave/test/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ = Describe("Wave owner references Suite", func() {
	var c client.Client
	var deployment *appsv1.Deployment
	var r *ReconcileDeployment
	var mgrStopped *sync.WaitGroup
	var stopMgr chan struct{}

	const timeout = time.Second * 5

	var cm1 *corev1.ConfigMap
	var cm2 *corev1.ConfigMap
	var s1 *corev1.Secret
	var s2 *corev1.Secret
	var ownerRef metav1.OwnerReference

	var create = func(obj object) {
		Expect(c.Create(context.TODO(), obj)).NotTo(HaveOccurred())
	}

	var update = func(obj object) {
		Expect(c.Update(context.TODO(), obj)).NotTo(HaveOccurred())
	}

	var get = func(obj object) {
		key := types.NamespacedName{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		}
		Eventually(func() error {
			return c.Get(context.TODO(), key, obj)
		}, timeout).Should(Succeed())
	}

	var getOwnerRef = func(deployment *appsv1.Deployment) metav1.OwnerReference {
		f := false
		t := true
		return metav1.OwnerReference{
			APIVersion:         "apps/v1",
			Kind:               "Deployment",
			Name:               deployment.Name,
			UID:                deployment.UID,
			Controller:         &f,
			BlockOwnerDeletion: &t,
		}
	}

	BeforeEach(func() {
		mgr, err := manager.New(cfg, manager.Options{})
		Expect(err).NotTo(HaveOccurred())
		c = mgr.GetClient()

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

		create(cm1)
		create(cm2)
		create(s1)
		create(s2)

		deployment = utils.ExampleDeployment.DeepCopy()
		create(deployment)

		ownerRef = getOwnerRef(deployment)

		stopMgr, mgrStopped = StartTestManager(mgr)
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

	Context("removeOwnerReferences", func() {
		BeforeEach(func() {
			for _, obj := range []object{cm1, cm2, s1, s2} {
				otherRef := ownerRef.DeepCopy()
				otherRef.UID = obj.GetUID()
				obj.SetOwnerReferences([]metav1.OwnerReference{ownerRef, *otherRef})
				update(obj)

				Eventually(func() error {
					key := types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
					err := c.Get(context.TODO(), key, obj)
					if err != nil {
						return err
					}
					if len(obj.GetOwnerReferences()) != 2 {
						return fmt.Errorf("OwnerReferences not updated")
					}
					return nil
				}, timeout).Should(Succeed())
			}

			children := []object{cm1, s1}
			err := r.removeOwnerReferences(deployment, children)
			Expect(err).NotTo(HaveOccurred())

			get(cm1)
			get(cm2)
			get(s1)
			get(s2)

			// Updates should propogate
			for _, obj := range []object{cm1, s1} {
				Eventually(func() error {
					key := types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
					err := c.Get(context.TODO(), key, obj)
					if err != nil {
						return err
					}
					if len(obj.GetOwnerReferences()) != 1 {
						return fmt.Errorf("OwnerReferences not updated")
					}
					return nil
				}, timeout).Should(Succeed())
			}
		})

		It("removes owner references from the list of children given", func() {
			Expect(cm1.GetOwnerReferences()).NotTo(ContainElement(ownerRef))
			Expect(s1.GetOwnerReferences()).NotTo(ContainElement(ownerRef))
		})

		It("doesn't remove owner references from children not listed", func() {
			Expect(cm2.GetOwnerReferences()).To(ContainElement(ownerRef))
			Expect(s2.GetOwnerReferences()).To(ContainElement(ownerRef))
		})

		It("doesn't remove owner references pointing to other owners", func() {
			Expect(cm1.GetOwnerReferences()).To(ContainElement(Not(Equal(ownerRef))))
			Expect(cm2.GetOwnerReferences()).To(ContainElement(Not(Equal(ownerRef))))
			Expect(s1.GetOwnerReferences()).To(ContainElement(Not(Equal(ownerRef))))
			Expect(s2.GetOwnerReferences()).To(ContainElement(Not(Equal(ownerRef))))
		})
	})

	Context("updateOwnerReferences", func() {
		BeforeEach(func() {
			for _, obj := range []object{cm2, s1, s2} {
				obj.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
				update(obj)

				Eventually(func() error {
					key := types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}
					err := c.Get(context.TODO(), key, obj)
					if err != nil {
						return err
					}
					if len(obj.GetOwnerReferences()) != 1 {
						return fmt.Errorf("OwnerReferences not updated")
					}
					return nil
				}, timeout).Should(Succeed())
			}

			existing := []object{cm2, s1, s2}
			current := []object{cm1, s1}
			err := r.updateOwnerReferences(deployment, existing, current)
			Expect(err).NotTo(HaveOccurred())

			get(cm1)
			get(cm2)
			get(s1)
			get(s2)
		})

		It("removes owner references from those not in current", func() {
			Expect(cm2.GetOwnerReferences()).NotTo(ContainElement(ownerRef))
			Expect(s2.GetOwnerReferences()).NotTo(ContainElement(ownerRef))
		})

		It("adds owner references to those in current", func() {
			Expect(cm1.GetOwnerReferences()).To(ContainElement(ownerRef))
			Expect(s1.GetOwnerReferences()).To(ContainElement(ownerRef))
		})
	})

	Context("updateOwnerReference", func() {
		BeforeEach(func() {
			// Add an OwnerReference to cm2
			cm2.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
			update(cm2)
			Eventually(func() error {
				key := types.NamespacedName{Namespace: cm2.GetNamespace(), Name: cm2.GetName()}
				err := c.Get(context.TODO(), key, cm2)
				if err != nil {
					return err
				}
				if len(cm2.GetOwnerReferences()) != 1 {
					return fmt.Errorf("OwnerReferences not updated")
				}
				return nil
			}, timeout).Should(Succeed())

			get(cm1)
			get(cm2)
		})

		It("adds an OwnerReference if not present", func() {
			// Add an OwnerReference to cm1
			otherRef := ownerRef
			otherRef.UID = cm1.GetUID()
			cm1.SetOwnerReferences([]metav1.OwnerReference{otherRef})
			update(cm1)
			Eventually(func() error {
				key := types.NamespacedName{Namespace: cm1.GetNamespace(), Name: cm1.GetName()}
				err := c.Get(context.TODO(), key, cm1)
				if err != nil {
					return err
				}
				if len(cm1.GetOwnerReferences()) != 2 {
					return fmt.Errorf("OwnerReferences not updated")
				}
				return nil
			}, timeout).Should(Succeed())

			get(cm1)
			Expect(r.updateOwnerReference(deployment, cm1)).NotTo(HaveOccurred())
			Eventually(func() error {
				key := types.NamespacedName{Namespace: cm1.GetNamespace(), Name: cm1.GetName()}
				err := c.Get(context.TODO(), key, cm1)
				if err != nil {
					return err
				}
				if len(cm1.GetOwnerReferences()) != 1 {
					return fmt.Errorf("OwnerReferences not updated")
				}
				return nil
			}, timeout).Should(Succeed())

			Expect(cm1.GetOwnerReferences()).Should(ContainElement(ownerRef))
		})

		It("doesn't update the child object if there is already and OwnerReference present", func() {
			// Add an OwnerReference to cm2
			cm2.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
			update(cm2)
			Eventually(func() error {
				key := types.NamespacedName{Namespace: cm2.GetNamespace(), Name: cm2.GetName()}
				err := c.Get(context.TODO(), key, cm2)
				if err != nil {
					return err
				}
				if len(cm2.GetOwnerReferences()) != 1 {
					return fmt.Errorf("OwnerReferences not updated")
				}
				return nil
			}, timeout).Should(Succeed())

			// Get the original version
			get(cm2)
			originalVersion := cm2.GetResourceVersion()
			Expect(r.updateOwnerReference(deployment, cm2)).NotTo(HaveOccurred())

			// Compare current version
			get(cm2)
			Expect(cm2.GetResourceVersion()).To(Equal(originalVersion))
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
})
