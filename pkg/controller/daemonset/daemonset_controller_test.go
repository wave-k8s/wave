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

package daemonset

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wave-k8s/wave/pkg/core"
	"github.com/wave-k8s/wave/test/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("DaemonSet controller Suite", func() {
	var c client.Client
	var m utils.Matcher

	var daemonset *appsv1.DaemonSet
	var requests <-chan reconcile.Request

	const timeout = time.Second * 5
	const consistentlyTimeout = time.Second

	var ownerRef metav1.OwnerReference
	var cm1 *corev1.ConfigMap
	var cm2 *corev1.ConfigMap
	var cm3 *corev1.ConfigMap
	var s1 *corev1.Secret
	var s2 *corev1.Secret
	var s3 *corev1.Secret

	const modified = "modified"

	var waitForDaemonSetReconciled = func(obj core.Object) {
		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			},
		}
		// wait for reconcile for creating the DaemonSet
		Eventually(requests, timeout).Should(Receive(Equal(request)))
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
		recFn, requests = SetupTestReconcile(newReconciler(mgr))
		Expect(add(mgr, recFn)).NotTo(HaveOccurred())

		testCtx, testCancel = context.WithCancel(context.Background())
		go Run(testCtx, mgr)

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
		m.Get(cm1, timeout).Should(Succeed())
		m.Get(cm2, timeout).Should(Succeed())
		m.Get(cm3, timeout).Should(Succeed())
		m.Get(s1, timeout).Should(Succeed())
		m.Get(s2, timeout).Should(Succeed())
		m.Get(s3, timeout).Should(Succeed())

		daemonset = utils.ExampleDaemonSet.DeepCopy()

		// Create a daemonset and wait for it to be reconciled
		m.Create(daemonset).Should(Succeed())
		waitForDaemonSetReconciled(daemonset)

		ownerRef = utils.GetOwnerRefDaemonSet(daemonset)
	})

	AfterEach(func() {
		// Make sure to delete any finalizers (if the daemonset exists)
		Eventually(func() error {
			key := types.NamespacedName{Namespace: daemonset.GetNamespace(), Name: daemonset.GetName()}
			err := c.Get(context.TODO(), key, daemonset)
			if err != nil && errors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			daemonset.SetFinalizers([]string{})
			return c.Update(context.TODO(), daemonset)
		}, timeout).Should(Succeed())

		Eventually(func() error {
			key := types.NamespacedName{Namespace: daemonset.GetNamespace(), Name: daemonset.GetName()}
			err := c.Get(context.TODO(), key, daemonset)
			if err != nil && errors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			if len(daemonset.GetFinalizers()) > 0 {
				return fmt.Errorf("Finalizers not upated")
			}
			return nil
		}, timeout).Should(Succeed())

		utils.DeleteAll(cfg, timeout,
			&appsv1.DaemonSetList{},
			&corev1.ConfigMapList{},
			&corev1.SecretList{},
			&corev1.EventList{},
		)

		testCancel()
	})

	Context("When a DaemonSet is reconciled", func() {
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

				m.Update(daemonset, addAnnotation).Should(Succeed())
				waitForDaemonSetReconciled(daemonset)

				// Get the updated DaemonSet
				m.Get(daemonset, timeout).Should(Succeed())
			})

			It("Adds OwnerReferences to all children", func() {
				for _, obj := range []core.Object{cm1, cm2, cm3, s1, s2, s3} {
					m.Get(obj, timeout).Should(Succeed())
					Eventually(obj, timeout).Should(utils.WithOwnerReferences(ContainElement(ownerRef)))
				}
			})

			It("Adds a finalizer to the DaemonSet", func() {
				Eventually(daemonset, timeout).Should(utils.WithFinalizers(ContainElement(core.FinalizerString)))
			})

			It("Adds a config hash to the Pod Template", func() {
				Eventually(daemonset, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(core.ConfigHashAnnotation)))
			})

			It("Sends an event when updating the hash", func() {
				eventMessage := func(event *corev1.Event) string {
					return event.Message
				}
				hashMessage := "Configuration hash updated to ebabf80ef45218b27078a41ca16b35a4f91cb5672f389e520ae9da6ee3df3b1c"
				Eventually(func() *corev1.EventList {
					events := &corev1.EventList{}
					m.Client.List(context.TODO(), events)
					return events
				}, timeout).Should(utils.WithItems(ContainElement(WithTransform(eventMessage, Equal(hashMessage)))))
			})

			Context("And a child is removed", func() {
				var originalHash string
				BeforeEach(func() {
					Eventually(daemonset, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(core.ConfigHashAnnotation)))
					originalHash = daemonset.Spec.Template.GetAnnotations()[core.ConfigHashAnnotation]

					// Remove "container2" which references Secret example2 and ConfigMap
					// example2
					removeContainer2 := func(obj client.Object) client.Object {
						ss, _ := obj.(*appsv1.DaemonSet)
						containers := ss.Spec.Template.Spec.Containers
						Expect(containers[0].Name).To(Equal("container1"))
						ss.Spec.Template.Spec.Containers = []corev1.Container{containers[0]}
						return ss
					}

					m.Update(daemonset, removeContainer2).Should(Succeed())
					waitForDaemonSetReconciled(daemonset)
					waitForDaemonSetReconciled(daemonset)

					// Get the updated DaemonSet
					m.Get(daemonset, timeout).Should(Succeed())
				})

				It("Removes the OwnerReference from the orphaned ConfigMap", func() {
					Eventually(cm2, timeout).ShouldNot(utils.WithOwnerReferences(ContainElement(ownerRef)))
				})

				It("Removes the OwnerReference from the orphaned Secret", func() {
					Eventually(s2, timeout).ShouldNot(utils.WithOwnerReferences(ContainElement(ownerRef)))
				})

				It("Updates the config hash in the Pod Template", func() {
					Eventually(daemonset, timeout).ShouldNot(utils.WithPodTemplateAnnotations(HaveKeyWithValue(core.ConfigHashAnnotation, originalHash)))
				})
			})

			Context("And a child is updated", func() {
				var originalHash string

				BeforeEach(func() {
					Eventually(daemonset, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(core.ConfigHashAnnotation)))
					originalHash = daemonset.Spec.Template.GetAnnotations()[core.ConfigHashAnnotation]
				})

				Context("A ConfigMap volume is updated", func() {
					BeforeEach(func() {
						modifyCM := func(obj client.Object) client.Object {
							cm, _ := obj.(*corev1.ConfigMap)
							cm.Data["key1"] = modified
							return cm
						}
						m.Update(cm1, modifyCM).Should(Succeed())

						waitForDaemonSetReconciled(daemonset)

						// Get the updated DaemonSet
						m.Get(daemonset, timeout).Should(Succeed())
					})

					It("Updates the config hash in the Pod Template", func() {
						Eventually(daemonset, timeout).ShouldNot(utils.WithAnnotations(HaveKeyWithValue(core.ConfigHashAnnotation, originalHash)))
					})
				})

				Context("A ConfigMap EnvSource is updated", func() {
					BeforeEach(func() {
						modifyCM := func(obj client.Object) client.Object {
							cm, _ := obj.(*corev1.ConfigMap)
							cm.Data["key1"] = modified
							return cm
						}
						m.Update(cm2, modifyCM).Should(Succeed())

						waitForDaemonSetReconciled(daemonset)

						// Get the updated DaemonSet
						m.Get(daemonset, timeout).Should(Succeed())
					})

					It("Updates the config hash in the Pod Template", func() {
						Eventually(daemonset, timeout).ShouldNot(utils.WithAnnotations(HaveKeyWithValue(core.ConfigHashAnnotation, originalHash)))
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
						m.Update(s1, modifyS).Should(Succeed())

						waitForDaemonSetReconciled(daemonset)

						// Get the updated DaemonSet
						m.Get(daemonset, timeout).Should(Succeed())
					})

					It("Updates the config hash in the Pod Template", func() {
						Eventually(daemonset, timeout).ShouldNot(utils.WithAnnotations(HaveKeyWithValue(core.ConfigHashAnnotation, originalHash)))
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
						m.Update(s2, modifyS).Should(Succeed())

						waitForDaemonSetReconciled(daemonset)

						// Get the updated DaemonSet
						m.Get(daemonset, timeout).Should(Succeed())
					})

					It("Updates the config hash in the Pod Template", func() {
						Eventually(daemonset, timeout).ShouldNot(utils.WithAnnotations(HaveKeyWithValue(core.ConfigHashAnnotation, originalHash)))
					})
				})
			})

			Context("And the annotation is removed", func() {
				BeforeEach(func() {
					removeAnnotations := func(obj client.Object) client.Object {
						obj.SetAnnotations(make(map[string]string))
						return obj
					}
					m.Update(daemonset, removeAnnotations).Should(Succeed())
					waitForDaemonSetReconciled(daemonset)
					waitForDaemonSetReconciled(daemonset)
					m.Get(daemonset).Should(Succeed())
					Eventually(daemonset, timeout).ShouldNot(utils.WithAnnotations(HaveKey(core.RequiredAnnotation)))
				})

				It("Removes the OwnerReference from the all children", func() {
					for _, obj := range []core.Object{cm1, cm2, s1, s2} {
						m.Get(obj, timeout).Should(Succeed())
						Eventually(obj, timeout).ShouldNot(utils.WithOwnerReferences(ContainElement(ownerRef)))
					}
				})

				It("Removes the DaemonSet's finalizer", func() {
					m.Get(daemonset).Should(Succeed())
					Eventually(daemonset, timeout).ShouldNot(utils.WithFinalizers(ContainElement(core.FinalizerString)))
				})
			})

			Context("And is deleted", func() {
				BeforeEach(func() {
					// Make sure the cache has synced before we run the test
					Eventually(daemonset, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(core.ConfigHashAnnotation)))
					m.Delete(daemonset).Should(Succeed())
					waitForDaemonSetReconciled(daemonset)

					// Get the updated DaemonSet
					m.Get(daemonset, timeout).Should(Succeed())
					Eventually(daemonset, timeout).ShouldNot(utils.WithDeletionTimestamp(BeNil()))
				})
				It("Removes the OwnerReference from the all children", func() {
					for _, obj := range []core.Object{cm1, cm2, s1, s2} {
						Eventually(obj, timeout).ShouldNot(utils.WithOwnerReferences(ContainElement(ownerRef)))
					}
				})

				It("Removes the DaemonSet's finalizer", func() {
					// Removing the finalizer causes the daemonset to be deleted
					m.Get(daemonset, timeout).ShouldNot(Succeed())
				})
			})
		})

		Context("And it does not have the required annotation", func() {
			BeforeEach(func() {
				// Get the updated DaemonSet
				m.Get(daemonset, timeout).Should(Succeed())
			})

			It("Doesn't add any OwnerReferences to any children", func() {
				for _, obj := range []core.Object{cm1, cm2, s1, s2} {
					m.Consistently(obj, consistentlyTimeout).ShouldNot(utils.WithOwnerReferences(ContainElement(ownerRef)))
				}
			})

			It("Doesn't add a finalizer to the DaemonSet", func() {
				m.Consistently(daemonset, consistentlyTimeout).ShouldNot(utils.WithFinalizers(ContainElement(core.FinalizerString)))
			})

			It("Doesn't add a config hash to the Pod Template", func() {
				m.Consistently(daemonset, consistentlyTimeout).ShouldNot(utils.WithAnnotations(ContainElement(core.ConfigHashAnnotation)))
			})
		})
	})

})
