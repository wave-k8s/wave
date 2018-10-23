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
		create(utils.ExampleConfigMap1.DeepCopy())
		create(utils.ExampleConfigMap2.DeepCopy())
		create(utils.ExampleSecret1.DeepCopy())
		create(utils.ExampleSecret2.DeepCopy())

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

	// Waiting for handleDelete to be implemented
	PContext("handleDelete", func() {
		var cm1 *corev1.ConfigMap
		var cm2 *corev1.ConfigMap
		var s1 *corev1.Secret
		var s2 *corev1.Secret

		BeforeEach(func() {
			cm1 = utils.ExampleConfigMap1.DeepCopy()
			cm2 = utils.ExampleConfigMap2.DeepCopy()
			s1 = utils.ExampleSecret1.DeepCopy()
			s2 = utils.ExampleSecret2.DeepCopy()

			for _, obj := range []object{cm1, cm2, s1, s2} {
				obj.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
				update(obj)
			}

			f := deployment.GetFinalizers()
			f = append(f, finalizerString)
			deployment.SetFinalizers(f)
			update(deployment)

			_, err := r.handleDelete(deployment)
			Expect(err).NotTo(HaveOccurred())
		})

		It("removes owner references from all children", func() {
			for _, obj := range []object{cm1, cm2, s1, s2} {
				get(obj)
				Expect(obj.GetOwnerReferences()).NotTo(ContainElement(ownerRef))
			}
		})

		It("removes the finalizer from the deployment", func() {
			Expect(deployment.GetFinalizers()).NotTo(ContainElement(finalizerString))
		})
	})

	// Waiting for toBeDeleted to be implemented
	Context("toBeDeleted", func() {
		It("returns true if deletion timestamp is non-nil", func() {
			t := metav1.NewTime(time.Now())
			deployment.SetDeletionTimestamp(&t)
			Expect(toBeDeleted(deployment)).To(BeTrue())
		})

		It("returns false if the deleteion timestamp is nil", func() {
			Expect(toBeDeleted(deployment)).To(BeFalse())
		})

	})

})
