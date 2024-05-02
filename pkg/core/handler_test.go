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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ = Describe("Wave controller Suite", func() {
	var c client.Client
	var h *Handler
	var m utils.Matcher

	var deployment *appsv1.Deployment
	var mgrStopped *sync.WaitGroup
	var stopMgr chan struct{}

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
	var example1Name types.NamespacedName
	var example2Name types.NamespacedName
	var example3Name types.NamespacedName
	var instanceName types.NamespacedName

	var modified = "modified"

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

		stopMgr, mgrStopped = StartTestManager(mgr)

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

		example1Name = types.NamespacedName{
			Name:      "example1",
			Namespace: deployment.GetNamespace(),
		}
		example2Name = types.NamespacedName{
			Name:      "example2",
			Namespace: deployment.GetNamespace(),
		}
		example3Name = types.NamespacedName{
			Name:      "example3",
			Namespace: deployment.GetNamespace(),
		}
		instanceName = types.NamespacedName{
			Name:      deployment.GetName(),
			Namespace: deployment.GetNamespace(),
		}

		// Create a deployment and wait for it to be reconciled
		m.Create(deployment).Should(Succeed())
		_, err = h.HandleDeployment(deployment)
		Expect(err).NotTo(HaveOccurred())

		m.Get(deployment).Should(Succeed())
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

	Context("When a Deployment is reconciled", func() {
		Context("And it has the required annotation", func() {
			BeforeEach(func() {
				annotations := deployment.GetAnnotations()
				if annotations == nil {
					annotations = make(map[string]string)
				}
				annotations[RequiredAnnotation] = requiredAnnotationValue

				m.Update(deployment, func(obj client.Object) client.Object {
					obj.SetAnnotations(annotations)
					return obj
				}, timeout).Should(Succeed())
				_, err := h.HandleDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				// Get the updated Deployment
				m.Get(deployment, timeout).Should(Succeed())
				_, err = h.HandleDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				// Get the updated Deployment
				m.Get(deployment, timeout).Should(Succeed())
			})

			It("Adds a config hash to the Pod Template", func() {
				Eventually(deployment, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(ConfigHashAnnotation)))
			})

			It("Is watched by the handler", func() {
				Expect(h.GetWatchedConfigmaps().watchers[example1Name]).To(HaveKey(instanceName))
				Expect(h.GetWatchedConfigmaps().watchers[example2Name]).To(HaveKey(instanceName))
				Expect(h.GetWatchedConfigmaps().watchers[example3Name]).To(HaveKey(instanceName))
				Expect(h.GetWatchedSecrets().watchers[example1Name]).To(HaveKey(instanceName))
				Expect(h.GetWatchedSecrets().watchers[example2Name]).To(HaveKey(instanceName))
				Expect(h.GetWatchedSecrets().watchers[example3Name]).To(HaveKey(instanceName))
			})

			It("Sends an event when updating the hash", func() {
				Eventually(deployment, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(ConfigHashAnnotation)))

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
					Eventually(deployment, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(ConfigHashAnnotation)))
					originalHash = deployment.Spec.Template.GetAnnotations()[ConfigHashAnnotation]

					// Remove "container2" which references Secret example2 and ConfigMap
					// example2
					containers := deployment.Spec.Template.Spec.Containers
					Expect(containers[0].Name).To(Equal("container1"))
					m.Update(deployment, func(obj client.Object) client.Object {
						dpl := obj.(*appsv1.Deployment)
						dpl.Spec.Template.Spec.Containers = []corev1.Container{containers[0]}
						return dpl
					}).Should(Succeed())
					_, err := h.HandleDeployment(deployment)
					Expect(err).NotTo(HaveOccurred())

					// Get the updated Deployment
					m.Get(deployment, timeout).Should(Succeed())
				})

				It("Updates the config hash in the Pod Template", func() {
					Eventually(func() string {
						return deployment.Spec.Template.GetAnnotations()[ConfigHashAnnotation]
					}, timeout).ShouldNot(Equal(originalHash))
				})

				It("Is is not longer watched by the handler", func() {
					Expect(h.GetWatchedConfigmaps().watchers[example1Name]).To(HaveKey(instanceName))
					Expect(h.GetWatchedConfigmaps().watchers[example2Name]).NotTo(HaveKey(instanceName))
					Expect(h.GetWatchedConfigmaps().watchers[example3Name]).To(HaveKey(instanceName))
					Expect(h.GetWatchedSecrets().watchers[example1Name]).To(HaveKey(instanceName))
					Expect(h.GetWatchedSecrets().watchers[example2Name]).NotTo(HaveKey(instanceName))
					Expect(h.GetWatchedSecrets().watchers[example3Name]).To(HaveKey(instanceName))
				})
			})

			Context("And a child is updated", func() {
				var originalHash string

				BeforeEach(func() {
					Eventually(deployment, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(ConfigHashAnnotation)))
					originalHash = deployment.Spec.Template.GetAnnotations()[ConfigHashAnnotation]
				})

				Context("A ConfigMap volume is updated", func() {
					BeforeEach(func() {
						m.Update(cm1, func(obj client.Object) client.Object {
							cm := obj.(*corev1.ConfigMap)
							cm.Data["key1"] = modified
							return cm
						}).Should(Succeed())

						_, err := h.HandleDeployment(deployment)
						Expect(err).NotTo(HaveOccurred())

						// Get the updated Deployment
						m.Get(deployment, timeout).Should(Succeed())
					})

					It("Updates the config hash in the Pod Template", func() {
						Eventually(func() string {
							return deployment.Spec.Template.GetAnnotations()[ConfigHashAnnotation]
						}, timeout).ShouldNot(Equal(originalHash))
					})
				})

				Context("A ConfigMap EnvSource is updated", func() {
					BeforeEach(func() {
						m.Update(cm2, func(obj client.Object) client.Object {
							cm := obj.(*corev1.ConfigMap)
							cm.Data["key1"] = modified
							return cm
						}, timeout).Should(Succeed())

						_, err := h.HandleDeployment(deployment)
						Expect(err).NotTo(HaveOccurred())

						// Get the updated Deployment
						m.Get(deployment, timeout).Should(Succeed())
					})

					It("Updates the config hash in the Pod Template", func() {
						Eventually(func() string {
							return deployment.Spec.Template.GetAnnotations()[ConfigHashAnnotation]
						}, timeout).ShouldNot(Equal(originalHash))
					})
				})

				Context("A ConfigMap Env for a key being used is updated", func() {
					BeforeEach(func() {
						m.Update(cm3, func(obj client.Object) client.Object {
							cm := obj.(*corev1.ConfigMap)
							cm.Data["key1"] = modified

							return cm
						}, timeout).Should(Succeed())

						_, err := h.HandleDeployment(deployment)
						Expect(err).NotTo(HaveOccurred())

						// Get the updated Deployment
						m.Get(deployment, timeout).Should(Succeed())
					})

					It("Updates the config hash in the Pod Template", func() {
						Eventually(func() string {
							return deployment.Spec.Template.GetAnnotations()[ConfigHashAnnotation]
						}, timeout).ShouldNot(Equal(originalHash))
					})
				})

				Context("A ConfigMap Env for a key that is not being used is updated", func() {
					BeforeEach(func() {
						m.Update(cm3, func(obj client.Object) client.Object {
							cm := obj.(*corev1.ConfigMap)
							cm.Data["key3"] = modified

							return cm
						}, timeout).Should(Succeed())

						_, err := h.HandleDeployment(deployment)
						Expect(err).NotTo(HaveOccurred())

						// Get the updated Deployment
						m.Get(deployment, timeout).Should(Succeed())
					})

					It("Does not update the config hash in the Pod Template", func() {
						m.Consistently(deployment, consistentlyTimeout).Should(utils.WithPodTemplateAnnotations(HaveKeyWithValue(ConfigHashAnnotation, originalHash)))
					})
				})

				Context("A Secret volume is updated", func() {
					BeforeEach(func() {
						m.Update(s1, func(obj client.Object) client.Object {
							s := obj.(*corev1.Secret)
							if s.StringData == nil {
								s.StringData = make(map[string]string)
							}
							s.StringData["key1"] = modified

							return s
						}, timeout).Should(Succeed())

						_, err := h.HandleDeployment(deployment)
						Expect(err).NotTo(HaveOccurred())

						// Get the updated Deployment
						m.Get(deployment, timeout).Should(Succeed())
					})

					It("Updates the config hash in the Pod Template", func() {
						Eventually(func() string {
							return deployment.Spec.Template.GetAnnotations()[ConfigHashAnnotation]
						}, timeout).ShouldNot(Equal(originalHash))
					})
				})

				Context("A Secret EnvSource is updated", func() {
					BeforeEach(func() {
						m.Update(s2, func(obj client.Object) client.Object {
							s := obj.(*corev1.Secret)
							if s.StringData == nil {
								s.StringData = make(map[string]string)
							}
							s.StringData["key1"] = modified

							return s
						}, timeout).Should(Succeed())

						_, err := h.HandleDeployment(deployment)
						Expect(err).NotTo(HaveOccurred())

						// Get the updated Deployment
						m.Get(deployment, timeout).Should(Succeed())
					})

					It("Updates the config hash in the Pod Template", func() {
						Eventually(func() string {
							return deployment.Spec.Template.GetAnnotations()[ConfigHashAnnotation]
						}, timeout).ShouldNot(Equal(originalHash))
					})
				})

				Context("A Secret Env for a key being used is updated", func() {
					BeforeEach(func() {
						m.Update(s3, func(obj client.Object) client.Object {
							s := obj.(*corev1.Secret)
							if s.StringData == nil {
								s.StringData = make(map[string]string)
							}
							s.StringData["key1"] = modified

							return s
						}, timeout).Should(Succeed())

						_, err := h.HandleDeployment(deployment)
						Expect(err).NotTo(HaveOccurred())

						// Get the updated Deployment
						m.Get(deployment, timeout).Should(Succeed())
					})

					It("Updates the config hash in the Pod Template", func() {
						Eventually(func() string {
							return deployment.Spec.Template.GetAnnotations()[ConfigHashAnnotation]
						}, timeout).ShouldNot(Equal(originalHash))
					})
				})

				Context("A Secret Env for a key that is not being used is updated", func() {
					BeforeEach(func() {
						m.Update(s3, func(obj client.Object) client.Object {
							s := obj.(*corev1.Secret)
							if s.StringData == nil {
								s.StringData = make(map[string]string)
							}
							s.StringData["key3"] = modified

							return s
						}, timeout).Should(Succeed())

						_, err := h.HandleDeployment(deployment)
						Expect(err).NotTo(HaveOccurred())

						// Get the updated Deployment
						m.Get(deployment, timeout).Should(Succeed())
					})

					It("Does not update the config hash in the Pod Template", func() {
						m.Consistently(deployment, consistentlyTimeout).Should(utils.WithPodTemplateAnnotations(HaveKeyWithValue(ConfigHashAnnotation, originalHash)))
					})
				})
			})

			Context("And the annotation is removed", func() {
				BeforeEach(func() {
					// Make sure the cache has synced before we run the test
					Eventually(deployment, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(ConfigHashAnnotation)))
					m.Update(deployment, func(obj client.Object) client.Object {
						obj.SetAnnotations(make(map[string]string))

						return obj
					}, timeout).Should(Succeed())
					_, err := h.HandleDeployment(deployment)
					Expect(err).NotTo(HaveOccurred())

					Eventually(deployment, timeout).ShouldNot(utils.WithAnnotations(HaveKey(RequiredAnnotation)))
				})

				It("No longer is watched by the handler", func() {
					Expect(len(h.GetWatchedConfigmaps().watchers)).To(Equal(0))
					Expect(len(h.GetWatchedSecrets().watchers)).To(Equal(0))
				})
			})

			Context("And is deleted", func() {
				BeforeEach(func() {
					// Make sure the cache has synced before we run the test
					Eventually(deployment, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(ConfigHashAnnotation)))

					m.Delete(deployment).Should(Succeed())

					// This test is suboptimal this the controller is not part of it here
					// Controller will call RemoveWatches if Delete cannot be found during reconcile
					h.RemoveWatches(instanceName)

				})
				It("Not longer exists", func() {
					m.Get(deployment).Should(MatchError(MatchRegexp(`not found`)))
				})

				It("No longer is watched by the handler", func() {
					Expect(len(h.GetWatchedConfigmaps().watchers)).To(Equal(0))
					Expect(len(h.GetWatchedSecrets().watchers)).To(Equal(0))
				})
			})
		})

		Context("And it does not have the required annotation", func() {
			BeforeEach(func() {
				// Get the updated Deployment
				m.Get(deployment, timeout).Should(Succeed())
			})

			It("Is not watched by the handler", func() {
				Expect(len(h.GetWatchedConfigmaps().watchers)).To(Equal(0))
				Expect(len(h.GetWatchedSecrets().watchers)).To(Equal(0))
			})

			It("Doesn't add a config hash to the Pod Template", func() {
				m.Consistently(deployment, consistentlyTimeout).ShouldNot(utils.WithAnnotations(ContainElement(ConfigHashAnnotation)))
			})

		})
	})

})
