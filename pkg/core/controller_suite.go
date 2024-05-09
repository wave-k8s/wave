package core

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wave-k8s/wave/test/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	webhook "sigs.k8s.io/controller-runtime/pkg/webhook"
)

func ControllerTestSuite[I InstanceType](
	t **envtest.Environment, cfg **rest.Config,
	makeObject func() I,
	startController func(mgr manager.Manager) (context.CancelFunc, chan reconcile.Request, chan reconcile.Request)) {
	var c client.Client
	var m utils.Matcher

	var testCancel context.CancelFunc

	var instance I
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

	var waitForInstanceReconciled = func(obj Object) {
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

	var consistentlyInstanceNotReconciled = func(obj Object) {
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

		mgr, err := manager.New(*cfg, manager.Options{
			Metrics: metricsserver.Options{
				BindAddress: "0",
			},
			WebhookServer: webhook.NewServer(webhook.Options{
				Host:    (*t).WebhookInstallOptions.LocalServingHost,
				Port:    (*t).WebhookInstallOptions.LocalServingPort,
				CertDir: (*t).WebhookInstallOptions.LocalServingCertDir,
			}),
		})
		Expect(err).NotTo(HaveOccurred())
		var cerr error
		c, cerr = client.New(*cfg, client.Options{Scheme: scheme.Scheme})
		Expect(cerr).NotTo(HaveOccurred())
		m = utils.Matcher{Client: c}

		testCancel, requestsStart, requests = startController(mgr)

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
		testCancel()

		utils.DeleteAll(*cfg, timeout,
			&appsv1.DaemonSetList{},
			&appsv1.DeploymentList{},
			&appsv1.StatefulSetList{},
			&corev1.ConfigMapList{},
			&corev1.SecretList{},
			&corev1.EventList{},
		)
	})

	Context("When a instance with all children existing is reconciled", func() {
		BeforeEach(func() {
			// Create a instance and wait for it to be reconciled
			clearReconciled()
			m.Create(instance).Should(Succeed())
			waitForInstanceReconciled(instance)
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
				clearReconciled()
				m.Update(instance, addAnnotation).Should(Succeed())
				waitForInstanceReconciled(instance)

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
					Eventually(instance, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(ConfigHashAnnotation)))
					originalHash = GetPodTemplate(instance).GetAnnotations()[ConfigHashAnnotation]

					// Remove "container2" which references Secret example2 and ConfigMap
					// example2
					clearReconciled()
					m.Get(instance, timeout).Should(Succeed())
					podTemplate := GetPodTemplate(instance)
					Expect(podTemplate.Spec.Containers[0].Name).To(Equal("container1"))
					podTemplate.Spec.Containers = []corev1.Container{podTemplate.Spec.Containers[0]}
					SetPodTemplate(instance, podTemplate)
					Expect(m.Client.Update(context.TODO(), instance)).Should(Succeed())
					waitForInstanceReconciled(instance)

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
					clearReconciled()

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
						clearReconciled()
						m.Update(cm1, modifyCM).Should(Succeed())
						waitForInstanceReconciled(instance)
						waitForInstanceReconciled(instance)

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
						clearReconciled()
						m.Update(cm2, modifyCM).Should(Succeed())
						waitForInstanceReconciled(instance)
						waitForInstanceReconciled(instance)

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
						clearReconciled()
						m.Update(s1, modifyS).Should(Succeed())
						waitForInstanceReconciled(instance)
						waitForInstanceReconciled(instance)

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
						clearReconciled()
						m.Update(s2, modifyS).Should(Succeed())
						waitForInstanceReconciled(instance)
						waitForInstanceReconciled(instance)

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
					clearReconciled()
					m.Update(instance, removeAnnotations).Should(Succeed())
					waitForInstanceReconciled(instance)
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
					clearReconciled()

					m.Update(cm1, modifyCM).Should(Succeed())
					consistentlyInstanceNotReconciled(instance)
				})
			})

			Context("And is deleted", func() {
				BeforeEach(func() {
					// Make sure the cache has synced before we run the test
					Eventually(instance, timeout).Should(utils.WithPodTemplateAnnotations(HaveKey(ConfigHashAnnotation)))
					clearReconciled()
					m.Delete(instance).Should(Succeed())
					waitForInstanceReconciled(instance)
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
					clearReconciled()

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
				clearReconciled()

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
			clearReconciled()
			m.Create(instance).Should(Succeed())
			waitForInstanceReconciled(instance)
		})

		It("Has scheduling disabled", func() {
			m.Get(instance, timeout).Should(Succeed())
			Expect(GetPodTemplate(instance).Spec.SchedulerName).To(Equal(SchedulingDisabledSchedulerName))
			Expect(instance.GetAnnotations()[SchedulingDisabledAnnotation]).To(Equal("default-scheduler"))
		})

		Context("And the missing child is created", func() {
			BeforeEach(func() {
				clearReconciled()
				cm1 = utils.ExampleConfigMap1.DeepCopy()
				m.Create(cm1).Should(Succeed())
				waitForInstanceReconciled(instance)
			})

			It("Has scheduling renabled", func() {
				m.Get(instance, timeout).Should(Succeed())
				Expect(GetPodTemplate(instance).Spec.SchedulerName).To(Equal("default-scheduler"))
				Expect(instance.GetAnnotations()).NotTo(HaveKey(SchedulingDisabledAnnotation))
			})
		})
	})

}
