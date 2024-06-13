package core

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wave-k8s/wave/test/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// SetupControllerTestReconcile returns a reconcile.Reconcile implementation that delegates to inner and
// writes the request to requests after Reconcile is finished.
func SetupControllerTestReconcile(inner reconcile.Reconciler) (reconcile.Reconciler, chan reconcile.Request, chan reconcile.Request) {
	requestsStart := make(chan reconcile.Request)
	requests := make(chan reconcile.Request)
	fn := reconcile.Func(func(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
		requestsStart <- req
		result, err := inner.Reconcile(ctx, req)
		requests <- req
		return result, err
	})
	return fn, requestsStart, requests
}

// Run runs the webhook server.
func Run(ctx context.Context, k8sManager ctrl.Manager) error {
	defer GinkgoRecover()
	if err := k8sManager.Start(ctx); err != nil {
		return err
	}
	return nil
}

func withTimeout(ch <-chan reconcile.Request, timeout time.Duration) (ok bool) {
	select {
	case <-ch:
		return true
	case <-time.After(timeout):
	}
	return false
}

func ControllerTestSuite[I InstanceType](
	t **envtest.Environment, cfg **rest.Config, mRef *utils.Matcher,
	requestsStart *<-chan reconcile.Request, requests *<-chan reconcile.Request,
	makeObject func() I) {

	var instance I
	var m utils.Matcher

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

	var waitForInstanceReconciled = func(obj Object, times int) {
		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			},
		}
		for range times {
			// wait for reconcile for creating the DaemonSet
			Eventually(*requestsStart, timeout).Should(Receive(Equal(request)))
			Eventually(*requests, timeout).Should(Receive(Equal(request)))
		}
	}

	var consistentlyInstanceNotReconciled = func(obj Object) {
		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			},
		}
		// wait for reconcile for creating the DaemonSet
		Consistently(*requestsStart, .1).ShouldNot(Receive(Equal(request)))
	}

	var expectNoReconciles = func() {
		consistentlyInstanceNotReconciled(instance)
	}

	BeforeEach(func() {
		m = *mRef

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

		instance = makeObject()
	})

	AfterEach(func() {
		expectNoReconciles()

		utils.DeleteAll(*cfg, timeout,
			&appsv1.DaemonSetList{},
			&appsv1.DeploymentList{},
			&appsv1.StatefulSetList{},
			&corev1.ConfigMapList{},
			&corev1.SecretList{},
			&corev1.EventList{},
		)
		// Let cleanup happen
		for {
			if ok := withTimeout(*requestsStart, time.Millisecond*100); ok {
				<-*requests
			} else {
				break
			}
		}
	})

	Context("When a instance with all children existing is reconciled", func() {
		BeforeEach(func() {
			// Create a instance and wait for it to be reconciled
			expectNoReconciles()
			m.Create(instance).Should(Succeed())
			waitForInstanceReconciled(instance, 1)
		})

		Context("And it has the required annotation", func() {
			BeforeEach(func() {
				addAnnotation := func(obj client.Object) client.Object {
					annotations := obj.GetAnnotations()
					if annotations == nil {
						annotations = make(map[string]string)
					}
					annotations[RequiredAnnotation] = "true"
					obj.SetAnnotations(annotations)
					return obj
				}
				expectNoReconciles()
				m.Update(instance, addAnnotation).Should(Succeed())
				waitForInstanceReconciled(instance, 1)

				// Get the updated instance
				m.Get(instance, timeout).Should(Succeed())
			})

			It("Has scheduling enabled", func() {
				m.Get(instance, timeout).Should(Succeed())
				Expect(GetPodTemplate(instance).Spec.SchedulerName).To(Equal("default-scheduler"))
				Expect(instance.GetAnnotations()).NotTo(HaveKey(SchedulingDisabledAnnotation))
			})

			It("Adds a config hash to the Pod Template", func() {
				Eventually(instance, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(ConfigHashAnnotation)))
			})

			It("Sends an event when updating the hash", func() {
				Eventually(instance, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(ConfigHashAnnotation)))

				eventMessage := func(event *corev1.Event) string {
					return event.Message
				}
				hashMessage := "Configuration hash updated to 318d4a3c6b9f6471f054001ea3103b2abb3693fe41922c733df45b53266d5216"
				Eventually(func() *corev1.EventList {
					events := &corev1.EventList{}
					Expect(m.Client.List(context.TODO(), events)).To(Succeed())
					return events
				}, timeout).Should(utils.WithItems(ContainElement(WithTransform(eventMessage, Equal(hashMessage)))))
			})

			Context("And a child is removed", func() {
				var originalHash string
				BeforeEach(func() {
					Eventually(instance, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(ConfigHashAnnotation)))
					originalHash = GetPodTemplate(instance).GetAnnotations()[ConfigHashAnnotation]

					// Remove "container2" which references Secret example2 and ConfigMap
					// example2
					expectNoReconciles()
					m.Get(instance, timeout).Should(Succeed())
					podTemplate := GetPodTemplate(instance)
					Expect(podTemplate.Spec.Containers[0].Name).To(Equal("container1"))
					podTemplate.Spec.Containers = []corev1.Container{podTemplate.Spec.Containers[0]}
					SetPodTemplate(instance, podTemplate)
					Expect(m.Client.Update(context.TODO(), instance)).Should(Succeed())
					waitForInstanceReconciled(instance, 1)

					// Get the updated instance
					m.Get(instance, timeout).Should(Succeed())
				})

				It("Updates the config hash in the Pod Template", func() {
					Eventually(func() string {
						return GetPodTemplate(instance).GetAnnotations()[ConfigHashAnnotation]
					}, timeout).ShouldNot(Equal(originalHash))
				})

				It("Changes to the removed children no longer trigger a reconcile", func() {
					modifyCM := func(obj client.Object) client.Object {
						cm, _ := obj.(*corev1.ConfigMap)
						cm.Data["key1"] = "modified"
						return cm
					}
					expectNoReconciles()

					m.Update(cm2, modifyCM).Should(Succeed())
					consistentlyInstanceNotReconciled(instance)
				})
			})

			Context("And a child is updated", func() {
				var originalHash string

				BeforeEach(func() {
					m.Get(instance, timeout).Should(Succeed())
					Eventually(instance, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(ConfigHashAnnotation)))
					originalHash = GetPodTemplate(instance).GetAnnotations()[ConfigHashAnnotation]
				})

				Context("A ConfigMap volume is updated", func() {
					BeforeEach(func() {
						modifyCM := func(obj client.Object) client.Object {
							cm, _ := obj.(*corev1.ConfigMap)
							cm.Data["key1"] = modified
							return cm
						}
						expectNoReconciles()
						m.Update(cm1, modifyCM).Should(Succeed())
						waitForInstanceReconciled(instance, 2) // Reschedules once since we update the hash

						// Get the updated instance
						m.Get(instance, timeout).Should(Succeed())
					})

					It("Updates the config hash in the Pod Template", func() {
						Eventually(func() string {
							return GetPodTemplate(instance).GetAnnotations()[ConfigHashAnnotation]
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
						expectNoReconciles()
						m.Update(cm2, modifyCM).Should(Succeed())
						waitForInstanceReconciled(instance, 2) // Reschedules once since we update the hash

						// Get the updated instance
						m.Get(instance, timeout).Should(Succeed())
					})

					It("Updates the config hash in the Pod Template", func() {
						Eventually(func() string {
							return GetPodTemplate(instance).GetAnnotations()[ConfigHashAnnotation]
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
						expectNoReconciles()
						m.Update(s1, modifyS).Should(Succeed())
						waitForInstanceReconciled(instance, 2) // Reschedules once since we update the hash

						// Get the updated instance
						m.Get(instance, timeout).Should(Succeed())
					})

					It("Updates the config hash in the Pod Template", func() {
						Eventually(func() string {
							return GetPodTemplate(instance).GetAnnotations()[ConfigHashAnnotation]
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
						expectNoReconciles()
						m.Update(s2, modifyS).Should(Succeed())
						waitForInstanceReconciled(instance, 2) // Reschedules once since we update the hash

						// Get the updated instance
						m.Get(instance, timeout).Should(Succeed())
					})

					It("Updates the config hash in the Pod Template", func() {
						Eventually(func() string {
							return GetPodTemplate(instance).GetAnnotations()[ConfigHashAnnotation]
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
					expectNoReconciles()
					m.Update(instance, removeAnnotations).Should(Succeed())
					waitForInstanceReconciled(instance, 1)
					m.Get(instance).Should(Succeed())
					Eventually(instance, timeout).ShouldNot(utils.WithAnnotations(HaveKey(RequiredAnnotation)))
				})

				It("Removes the config hash annotation", func() {
					m.Consistently(instance, consistentlyTimeout).ShouldNot(utils.WithAnnotations(ContainElement(ConfigHashAnnotation)))
				})

				It("Changes to children no longer trigger a reconcile", func() {
					modifyCM := func(obj client.Object) client.Object {
						cm, _ := obj.(*corev1.ConfigMap)
						cm.Data["key1"] = "modified"
						return cm
					}
					expectNoReconciles()

					m.Update(cm1, modifyCM).Should(Succeed())
					consistentlyInstanceNotReconciled(instance)
				})
			})

			Context("And is deleted", func() {
				BeforeEach(func() {
					// Make sure the cache has synced before we run the test
					Eventually(instance, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(ConfigHashAnnotation)))
					expectNoReconciles()
					m.Delete(instance).Should(Succeed())
					waitForInstanceReconciled(instance, 1)
				})
				It("Not longer exists", func() {
					m.Get(instance).Should(MatchError(MatchRegexp(`not found`)))
				})

				It("Changes to children no longer trigger a reconcile", func() {
					modifyCM := func(obj client.Object) client.Object {
						cm, _ := obj.(*corev1.ConfigMap)
						cm.Data["key1"] = "modified"
						return cm
					}
					expectNoReconciles()

					m.Update(cm1, modifyCM).Should(Succeed())
					consistentlyInstanceNotReconciled(instance)
				})
			})
		})

		Context("And it does not have the required annotation", func() {
			BeforeEach(func() {
				// Get the updated instance
				m.Get(instance, timeout).Should(Succeed())
			})

			It("Doesn't add a config hash to the Pod Template", func() {
				m.Consistently(instance, consistentlyTimeout).ShouldNot(utils.WithAnnotations(ContainElement(ConfigHashAnnotation)))
			})

			It("Changes to children no do not trigger a reconcile", func() {
				modifyCM := func(obj client.Object) client.Object {
					cm, _ := obj.(*corev1.ConfigMap)
					cm.Data["key1"] = "modified"
					return cm
				}
				expectNoReconciles()

				m.Update(cm1, modifyCM).Should(Succeed())
				consistentlyInstanceNotReconciled(instance)
			})
		})
	})

	Context("When a instance with missing children is reconciled", func() {
		BeforeEach(func() {
			m.Delete(cm1).Should(Succeed())

			annotations := instance.GetAnnotations()
			if annotations == nil {
				annotations = make(map[string]string)
			}
			annotations[RequiredAnnotation] = "true"
			instance.SetAnnotations(annotations)

			// Create a instance and wait for it to be reconciled
			expectNoReconciles()
			m.Create(instance).Should(Succeed())
			waitForInstanceReconciled(instance, 1)
		})

		It("Has scheduling disabled", func() {
			m.Get(instance, timeout).Should(Succeed())
			Expect(GetPodTemplate(instance).Spec.SchedulerName).To(Equal(SchedulingDisabledSchedulerName))
			Expect(instance.GetAnnotations()[SchedulingDisabledAnnotation]).To(Equal("default-scheduler"))
		})

		Context("And the missing child is created", func() {
			BeforeEach(func() {
				expectNoReconciles()
				cm1 = utils.ExampleConfigMap1.DeepCopy()
				m.Create(cm1).Should(Succeed())
				waitForInstanceReconciled(instance, 2) // Two since updating the scheduler self-triggers
			})

			It("Has scheduling renabled", func() {
				m.Get(instance, timeout).Should(Succeed())
				Expect(GetPodTemplate(instance).Spec.SchedulerName).To(Equal("default-scheduler"))
				Expect(instance.GetAnnotations()).NotTo(HaveKey(SchedulingDisabledAnnotation))
			})
		})

	})

	Context("When a instance with missing children in projection is reconciled", func() {
		BeforeEach(func() {
			m.Delete(cm6).Should(Succeed())

			annotations := instance.GetAnnotations()
			if annotations == nil {
				annotations = make(map[string]string)
			}
			annotations[RequiredAnnotation] = "true"
			instance.SetAnnotations(annotations)

			// Create a instance and wait for it to be reconciled
			expectNoReconciles()
			m.Create(instance).Should(Succeed())
			waitForInstanceReconciled(instance, 1)
		})

		It("Has scheduling disabled", func() {
			m.Get(instance, timeout).Should(Succeed())
			Expect(GetPodTemplate(instance).Spec.SchedulerName).To(Equal(SchedulingDisabledSchedulerName))
			Expect(instance.GetAnnotations()[SchedulingDisabledAnnotation]).To(Equal("default-scheduler"))
		})

		Context("And the missing child is created with a missing field", func() {
			BeforeEach(func() {
				expectNoReconciles()
				cm6Alt := utils.ExampleConfigMap6WithoutKey3.DeepCopy()
				m.Create(cm6Alt).Should(Succeed())
				waitForInstanceReconciled(instance, 1)
			})
			It("Has Scheduling still disabled", func() {
				m.Get(instance, timeout).Should(Succeed())
				Expect(GetPodTemplate(instance).Spec.SchedulerName).To(Equal(SchedulingDisabledSchedulerName))
				Expect(instance.GetAnnotations()[SchedulingDisabledAnnotation]).To(Equal("default-scheduler"))
			})
			Context("And the missing field is added", func() {
				BeforeEach(func() {
					expectNoReconciles()
					m.Get(cm6, timeout).Should(Succeed())
					cm6Copy := utils.ExampleConfigMap6.DeepCopy()
					cm6.Data = cm6Copy.Data
					Expect(m.Client.Update(context.TODO(), cm6)).Should(Succeed())
					waitForInstanceReconciled(instance, 2) // Reschedules once since we update the hash + reenables scheduling
				})

				It("Has scheduling renabled", func() {
					m.Get(instance, timeout).Should(Succeed())
					Expect(GetPodTemplate(instance).Spec.SchedulerName).To(Equal("default-scheduler"))
					Expect(instance.GetAnnotations()).NotTo(HaveKey(SchedulingDisabledAnnotation))
				})
			})

		})
	})

}
