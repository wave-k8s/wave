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

var _ = Describe("Wave children Suite", func() {
	var c client.Client
	var deployment *appsv1.Deployment
	var r *ReconcileDeployment
	var children []metav1.Object
	var mgrStopped *sync.WaitGroup
	var stopMgr chan struct{}

	const timeout = time.Second * 5

	var create = func(obj object) {
		Expect(c.Create(context.TODO(), obj)).NotTo(HaveOccurred())
	}

	var update = func(obj object) {
		Expect(c.Update(context.TODO(), obj)).NotTo(HaveOccurred())
	}

	var delete = func(obj object) {
		Expect(c.Delete(context.TODO(), obj)).NotTo(HaveOccurred())
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

	// Waiting for getCurrentChildren to be implemented
	PContext("getCurrentChildren", func() {
		BeforeEach(func() {
			var err error
			children, err = r.getCurrentChildren(deployment)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns ConfigMaps referenced in Volumes", func() {
			cm := utils.ExampleConfigMap1.DeepCopy()
			get(cm)
			Expect(children).To(ContainElement(cm))
		})

		It("returns ConfigMaps referenced in EnvFromSource", func() {
			cm := utils.ExampleConfigMap2.DeepCopy()
			get(cm)
			Expect(children).To(ContainElement(cm))
		})

		It("returns Secrets referenced in Volumes", func() {
			s := utils.ExampleSecret1.DeepCopy()
			get(s)
			Expect(children).To(ContainElement(s))
		})

		It("returns Secrets referenced in EnvFromSource", func() {
			s := utils.ExampleSecret2.DeepCopy()
			get(s)
			Expect(children).To(ContainElement(s))
		})

		It("does not return duplicate children", func() {
			Expect(children).To(HaveLen(4))
		})

		It("returns an error if one of the referenced children is missing", func() {
			s := utils.ExampleSecret2.DeepCopy()
			delete(s)

			current, err := r.getCurrentChildren(deployment)
			Expect(err).To(HaveOccurred())
			Expect(current).To(BeEmpty())
		})
	})

	// Waiting for getCurrentChildren to be implemented
	PContext("getExistingChildren", func() {
		BeforeEach(func() {
			get(deployment)
			ownerRef := getOwnerRef(deployment)

			cm := utils.ExampleConfigMap1.DeepCopy()
			s := utils.ExampleSecret1.DeepCopy()

			for _, obj := range []object{cm, s} {
				get(obj)
				obj.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
				update(obj)
			}

			var err error
			children, err = r.getCurrentChildren(deployment)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns ConfigMaps with the correct OwnerReference", func() {
			cm := utils.ExampleConfigMap1.DeepCopy()
			get(cm)
			Expect(children).To(ContainElement(cm))
		})

		It("doesn't return ConfigMaps without OwnerReferences", func() {
			cm := utils.ExampleConfigMap2.DeepCopy()
			get(cm)
			Expect(children).NotTo(ContainElement(cm))
		})

		It("returns Secrets with the correct OwnerReference", func() {
			s := utils.ExampleSecret1.DeepCopy()
			get(s)
			Expect(children).To(ContainElement(s))
		})

		It("doesn't return Secrets without OwnerReferences", func() {
			s := utils.ExampleSecret2.DeepCopy()
			get(s)
			Expect(children).NotTo(ContainElement(s))
		})

		It("does not return duplicate children", func() {
			Expect(children).To(HaveLen(2))
		})
	})

})
