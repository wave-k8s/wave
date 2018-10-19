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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type object interface {
	runtime.Object
	metav1.Object
}

var c client.Client

var deployment *appsv1.Deployment
var requests <-chan reconcile.Request
var mgrStopped *sync.WaitGroup
var stopMgr chan struct{}

const timeout = time.Second * 5

var _ = Describe("Wave controller Suite", func() {
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

	var waitForDeploymentReconciled = func(obj object) {
		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			},
		}
		// wait for reconcile for creating the Deployment
		Eventually(requests, timeout).Should(Receive(Equal(request)))
	}

	BeforeEach(func() {
		mgr, err := manager.New(cfg, manager.Options{})
		Expect(err).NotTo(HaveOccurred())
		c = mgr.GetClient()

		var recFn reconcile.Reconciler
		recFn, requests = SetupTestReconcile(newReconciler(mgr))
		Expect(add(mgr, recFn)).NotTo(HaveOccurred())

		stopMgr, mgrStopped = StartTestManager(mgr)

		// Create some configmaps and secrets
		create(utils.ExampleConfigMap1.DeepCopy())
		create(utils.ExampleConfigMap2.DeepCopy())
		create(utils.ExampleSecret1.DeepCopy())
		create(utils.ExampleSecret2.DeepCopy())

		deployment = utils.ExampleDeployment.DeepCopy()
		// Create a deployment and wait for it to be reconciled
		create(deployment)
		waitForDeploymentReconciled(deployment)
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

	Context("When a Deployment is reconciled", func() {
		Context("And it has the required annotation", func() {
			BeforeEach(func() {
				annotations := deployment.GetAnnotations()
				if annotations == nil {
					annotations = make(map[string]string)
				}
				annotations["wave.pusher.com/update-on-config-change"] = "true"
				deployment.SetAnnotations(annotations)

				update(deployment)
				waitForDeploymentReconciled(deployment)

				// Get the updated Deployment
				get(deployment)
			})

			It("Adds OwnerReferences to all children", func() {})

			It("Adds a finalizer to the Deployment", func() {})

			It("Adds a config hash to the Pod Template", func() {})

			Context("And a child is removed", func() {
				BeforeEach(func() {
					// Remove "container2" which references Secret example2 and ConfigMap
					// example2
					containers := deployment.Spec.Template.Spec.Containers
					Expect(containers[0].Name).To(Equal("container1"))
					deployment.Spec.Template.Spec.Containers = []corev1.Container{containers[0]}
					update(deployment)
					waitForDeploymentReconciled(deployment)

					// Get the updated Deployment
					get(deployment)
				})

				It("Removes the OwnerReference from the orphaned child", func() {})

				It("Updates the config hash in the Pod Template", func() {})
			})

			// TODO: Pending while Owner References not added to children
			PContext("And a child is updated", func() {
				Context("A ConfigMap volume is updated", func() {
					BeforeEach(func() {
						cm := utils.ExampleConfigMap1.DeepCopy()
						cm.Data["key1"] = "modified"
						update(cm)

						waitForDeploymentReconciled(deployment)

						// Get the updated Deployment
						get(deployment)
					})

					It("Updates the config hash in the Pod Template", func() {})
				})

				Context("A ConfigMap EnvSource is updated", func() {
					BeforeEach(func() {
						cm := utils.ExampleConfigMap2.DeepCopy()
						cm.Data["key1"] = "modified"
						update(cm)

						waitForDeploymentReconciled(deployment)

						// Get the updated Deployment
						get(deployment)
					})

					It("Updates the config hash in the Pod Template", func() {})
				})

				Context("A Secret volume is updated", func() {
					BeforeEach(func() {
						s := utils.ExampleSecret1.DeepCopy()
						s.StringData["key1"] = "modified"
						update(s)

						waitForDeploymentReconciled(deployment)

						// Get the updated Deployment
						get(deployment)
					})

					It("Updates the config hash in the Pod Template", func() {})
				})

				Context("A Secret EnvSource is updated", func() {
					BeforeEach(func() {
						s := utils.ExampleSecret2.DeepCopy()
						s.StringData["key1"] = "modified"
						update(s)

						waitForDeploymentReconciled(deployment)

						// Get the updated Deployment
						get(deployment)
					})

					It("Updates the config hash in the Pod Template", func() {})
				})
			})

			// TODO: Pending while finalizer not added to deployment
			PContext("And is deleted", func() {
				BeforeEach(func() {
					delete(deployment)
					waitForDeploymentReconciled(deployment)

					// Get the updated Deployment
					get(deployment)
				})
				It("Removes the OwnerReference from the all children", func() {})

				It("Removes the Deployment's finalizer", func() {})
			})
		})

		Context("And it does not have the required annotation", func() {
			BeforeEach(func() {
				// Get the updated Deployment
				get(deployment)
			})

			It("Doesn't add any OwnerReferences to any children", func() {})

			It("Doesn't add a finalizer to the Deployment", func() {})

			It("Doesn't add a config hash to the Pod Template", func() {})
		})
	})

})
