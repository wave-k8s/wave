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

package deployment

import (
	"context"
	"time"

	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wave-k8s/wave/pkg/core"
	"github.com/wave-k8s/wave/test/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Deployment controller Suite", func() {
	var c client.Client
	var m utils.Matcher

	var deployment *appsv1.Deployment
	var requestsStart <-chan reconcile.Request
	var requests <-chan reconcile.Request

	const timeout = time.Second * 5
	const consistentlyTimeout = time.Second

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

	const modified = "modified"

	var waitForDeploymentReconciled = func(obj core.Object) {
		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			},
		}
		// wait for reconcile for creating the DaemonSet
		Eventually(requestsStart, timeout).Should(Receive(Equal(request)))
		Eventually(requests, timeout).Should(Receive(Equal(request)))
	}

	var consistentlyDeploymentNotReconciled = func(obj core.Object) {
		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			},
		}
		// wait for reconcile for creating the DaemonSet
		Consistently(requestsStart, consistentlyTimeout).ShouldNot(Receive(Equal(request)))
	}

	var clearReconciled = func() {
		for len(requestsStart) > 0 {
			<-requestsStart
			<-requests
		}
	}

	BeforeEach(func() {
		// Reset the Prometheus Registry before each test to avoid errors
		metrics.Registry = prometheus.NewRegistry()

		mgr, err := manager.New(cfg, manager.Options{
			Metrics: metricsserver.Options{
				BindAddress: "0",
			},
		})
		Expect(err).NotTo(HaveOccurred())
		var cerr error
		c, cerr = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(cerr).NotTo(HaveOccurred())
		m = utils.Matcher{Client: c}

		var recFn reconcile.Reconciler
		r := newReconciler(mgr)
		recFn, requestsStart, requests = SetupTestReconcile(r)
		Expect(add(mgr, recFn, r.handler)).NotTo(HaveOccurred())

		testCtx, testCancel = context.WithCancel(context.Background())
		go Run(testCtx, mgr)

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

		deployment = utils.ExampleDeployment.DeepCopy()

		// Create a deployment and wait for it to be reconciled
		clearReconciled()
		m.Create(deployment).Should(Succeed())
		waitForDeploymentReconciled(deployment)
	})

	AfterEach(func() {
		testCancel()

		utils.DeleteAll(cfg, timeout,
			&appsv1.DeploymentList{},
			&corev1.ConfigMapList{},
			&corev1.SecretList{},
			&corev1.EventList{},
		)
	})

	Context("When a Deployment is reconciled", func() {
		Context("And it has the required annotation", func() {
			BeforeEach(func() {
				addAnnotation := func(obj client.Object) client.Object {
					annotations := obj.GetAnnotations()
					if annotations == nil {
						annotations = make(map[string]string)
					}
					annotations[core.RequiredAnnotation] = "true"
					obj.SetAnnotations(annotations)
					return obj
				}
				clearReconciled()
				m.Update(deployment, addAnnotation).Should(Succeed())
				// Two runs since we the controller retriggers itself by changing the object
				waitForDeploymentReconciled(deployment)
				waitForDeploymentReconciled(deployment)

				// Get the updated Deployment
				m.Get(deployment, timeout).Should(Succeed())
			})

			It("Adds a config hash to the Pod Template", func() {
				Eventually(deployment, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(core.ConfigHashAnnotation)))
			})

			It("Sends an event when updating the hash", func() {
				Eventually(deployment, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(core.ConfigHashAnnotation)))

				eventMessage := func(event *corev1.Event) string {
					return event.Message
				}
				hashMessage := "Configuration hash updated to 421778c325761f51dbf7a23a20eb9c1bc516ffd4aa7362ebec03175d427d7557"
				Eventually(func() *corev1.EventList {
					events := &corev1.EventList{}
					m.Client.List(context.TODO(), events)
					return events
				}, timeout).Should(utils.WithItems(ContainElement(WithTransform(eventMessage, Equal(hashMessage)))))
			})

			Context("And a child is removed", func() {
				var originalHash string
				BeforeEach(func() {
					Eventually(deployment, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(core.ConfigHashAnnotation)))
					originalHash = deployment.Spec.Template.GetAnnotations()[core.ConfigHashAnnotation]

					// Remove "container2" which references Secret example2 and ConfigMap
					// example2
					removeContainer2 := func(obj client.Object) client.Object {
						dep, _ := obj.(*appsv1.Deployment)
						containers := dep.Spec.Template.Spec.Containers
						Expect(containers[0].Name).To(Equal("container1"))
						dep.Spec.Template.Spec.Containers = []corev1.Container{containers[0]}
						return dep
					}
					clearReconciled()
					m.Update(deployment, removeContainer2).Should(Succeed())
					waitForDeploymentReconciled(deployment)
					waitForDeploymentReconciled(deployment)

					// Get the updated Deployment
					m.Get(deployment, timeout).Should(Succeed())
				})

				It("Updates the config hash in the Pod Template", func() {
					Eventually(func() string {
						return deployment.Spec.Template.GetAnnotations()[core.ConfigHashAnnotation]
					}, timeout).ShouldNot(Equal(originalHash))
				})

				It("Changes to the removed children no longer trigger a reconcile", func() {
					modifyCM := func(obj client.Object) client.Object {
						cm, _ := obj.(*corev1.ConfigMap)
						cm.Data["key1"] = "modified"
						return cm
					}
					clearReconciled()

					m.Update(cm2, modifyCM).Should(Succeed())
					consistentlyDeploymentNotReconciled(deployment)
				})
			})

			Context("And a child is updated", func() {
				var originalHash string

				BeforeEach(func() {
					m.Get(deployment, timeout).Should(Succeed())
					Eventually(deployment, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(core.ConfigHashAnnotation)))
					originalHash = deployment.Spec.Template.GetAnnotations()[core.ConfigHashAnnotation]
				})

				Context("A ConfigMap volume is updated", func() {
					BeforeEach(func() {
						modifyCM := func(obj client.Object) client.Object {
							cm, _ := obj.(*corev1.ConfigMap)
							cm.Data["key1"] = modified
							return cm
						}
						clearReconciled()
						m.Update(cm1, modifyCM).Should(Succeed())
						waitForDeploymentReconciled(deployment)
						waitForDeploymentReconciled(deployment)

						// Get the updated Deployment
						m.Get(deployment, timeout).Should(Succeed())
					})

					It("Updates the config hash in the Pod Template", func() {
						Eventually(func() string {
							return deployment.Spec.Template.GetAnnotations()[core.ConfigHashAnnotation]
						}, timeout).ShouldNot(Equal(originalHash))
					})
				})

				Context("A ConfigMap EnvSource is updated", func() {
					BeforeEach(func() {
						modifyCM := func(obj client.Object) client.Object {
							cm, _ := obj.(*corev1.ConfigMap)
							cm.Data["key1"] = modified
							return cm
						}
						clearReconciled()
						m.Update(cm2, modifyCM).Should(Succeed())
						waitForDeploymentReconciled(deployment)
						waitForDeploymentReconciled(deployment)

						// Get the updated Deployment
						m.Get(deployment, timeout).Should(Succeed())
					})

					It("Updates the config hash in the Pod Template", func() {
						Eventually(func() string {
							return deployment.Spec.Template.GetAnnotations()[core.ConfigHashAnnotation]
						}, timeout).ShouldNot(Equal(originalHash))
					})
				})

				Context("A Secret volume is updated", func() {
					BeforeEach(func() {
						modifyS := func(obj client.Object) client.Object {
							s, _ := obj.(*corev1.Secret)
							if s.StringData == nil {
								s.StringData = make(map[string]string)
							}
							s.StringData["key1"] = modified
							return s
						}
						clearReconciled()
						m.Update(s1, modifyS).Should(Succeed())
						waitForDeploymentReconciled(deployment)
						waitForDeploymentReconciled(deployment)

						// Get the updated Deployment
						m.Get(deployment, timeout).Should(Succeed())
					})

					It("Updates the config hash in the Pod Template", func() {
						Eventually(func() string {
							return deployment.Spec.Template.GetAnnotations()[core.ConfigHashAnnotation]
						}, timeout).ShouldNot(Equal(originalHash))
					})
				})

				Context("A Secret EnvSource is updated", func() {
					BeforeEach(func() {
						modifyS := func(obj client.Object) client.Object {
							s, _ := obj.(*corev1.Secret)
							if s.StringData == nil {
								s.StringData = make(map[string]string)
							}
							s.StringData["key1"] = modified
							return s
						}
						clearReconciled()
						m.Update(s2, modifyS).Should(Succeed())
						waitForDeploymentReconciled(deployment)
						waitForDeploymentReconciled(deployment)

						// Get the updated Deployment
						m.Get(deployment, timeout).Should(Succeed())
					})

					It("Updates the config hash in the Pod Template", func() {
						Eventually(func() string {
							return deployment.Spec.Template.GetAnnotations()[core.ConfigHashAnnotation]
						}, timeout).ShouldNot(Equal(originalHash))
					})
				})
			})

			Context("And the annotation is removed", func() {
				BeforeEach(func() {
					removeAnnotations := func(obj client.Object) client.Object {
						obj.SetAnnotations(make(map[string]string))
						return obj
					}
					clearReconciled()
					m.Update(deployment, removeAnnotations).Should(Succeed())
					waitForDeploymentReconciled(deployment)
					m.Get(deployment).Should(Succeed())
					Eventually(deployment, timeout).ShouldNot(utils.WithAnnotations(HaveKey(core.RequiredAnnotation)))
				})

				It("Removes the config hash annotation", func() {
					m.Consistently(deployment, consistentlyTimeout).ShouldNot(utils.WithAnnotations(ContainElement(core.ConfigHashAnnotation)))
				})

				It("Changes to children no longer trigger a reconcile", func() {
					modifyCM := func(obj client.Object) client.Object {
						cm, _ := obj.(*corev1.ConfigMap)
						cm.Data["key1"] = "modified"
						return cm
					}
					clearReconciled()

					m.Update(cm1, modifyCM).Should(Succeed())
					consistentlyDeploymentNotReconciled(deployment)
				})
			})

			Context("And is deleted", func() {
				BeforeEach(func() {
					// Make sure the cache has synced before we run the test
					Eventually(deployment, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(core.ConfigHashAnnotation)))
					clearReconciled()
					m.Delete(deployment).Should(Succeed())
					waitForDeploymentReconciled(deployment)
				})
				It("Not longer exists", func() {
					m.Get(deployment).Should(MatchError(MatchRegexp(`not found`)))
				})

				It("Changes to children no longer trigger a reconcile", func() {
					modifyCM := func(obj client.Object) client.Object {
						cm, _ := obj.(*corev1.ConfigMap)
						cm.Data["key1"] = "modified"
						return cm
					}
					clearReconciled()

					m.Update(cm1, modifyCM).Should(Succeed())
					consistentlyDeploymentNotReconciled(deployment)
				})
			})
		})

		Context("And it does not have the required annotation", func() {
			BeforeEach(func() {
				// Get the updated Deployment
				m.Get(deployment, timeout).Should(Succeed())
			})

			It("Doesn't add a config hash to the Pod Template", func() {
				m.Consistently(deployment, consistentlyTimeout).ShouldNot(utils.WithAnnotations(ContainElement(core.ConfigHashAnnotation)))
			})

			It("Changes to children no do not trigger a reconcile", func() {
				modifyCM := func(obj client.Object) client.Object {
					cm, _ := obj.(*corev1.ConfigMap)
					cm.Data["key1"] = "modified"
					return cm
				}
				clearReconciled()

				m.Update(cm1, modifyCM).Should(Succeed())
				consistentlyDeploymentNotReconciled(deployment)
			})
		})
	})

})
