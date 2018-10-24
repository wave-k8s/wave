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
			}

			children := []metav1.Object{cm1, s1}
			err := r.removeOwnerReferences(deployment, children)
			Expect(err).NotTo(HaveOccurred())

			get(cm1)
			get(cm2)
			get(s1)
			get(s2)
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

	// Waiting for updateOwnerRefernces to be implemented
	PContext("updateOwnerReferences", func() {
		BeforeEach(func() {
			for _, obj := range []object{cm2, s1, s2} {
				obj.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
				update(obj)
			}

			existing := []metav1.Object{cm2, s1, s2}
			current := []metav1.Object{cm1, s1}
			err := r.updateOwnerReferences(deployment, existing, current)
			Expect(err).NotTo(HaveOccurred())
		})

		It("removes owner references from those not in current", func() {
			get(cm2)
			Expect(cm2.GetOwnerReferences()).NotTo(ContainElement(ownerRef))
			get(s2)
			Expect(s2.GetOwnerReferences()).NotTo(ContainElement(ownerRef))
		})

		It("adds owner references to those in current", func() {
			get(cm1)
			Expect(cm1.GetOwnerReferences()).To(ContainElement(ownerRef))
			get(s1)
			Expect(s1.GetOwnerReferences()).To(ContainElement(ownerRef))
		})
	})

})
