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
	"sync"
	"time"

	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wave-k8s/wave/test/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ = Describe("Wave migration Suite", func() {
	var c client.Client
	var h *Handler[*appsv1.Deployment]
	var m utils.Matcher
	var deploymentObject *appsv1.Deployment
	var mgrStopped *sync.WaitGroup
	var stopMgr chan struct{}

	const timeout = time.Second * 5

	var ownerRef metav1.OwnerReference

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
		h = NewHandler[*appsv1.Deployment](c, mgr.GetEventRecorderFor("wave"))
		m = utils.Matcher{Client: c}

		// Create some configmaps and secrets
		m.Create(utils.ExampleConfigMap1.DeepCopy()).Should(Succeed())
		m.Create(utils.ExampleConfigMap2.DeepCopy()).Should(Succeed())
		m.Create(utils.ExampleSecret1.DeepCopy()).Should(Succeed())
		m.Create(utils.ExampleSecret2.DeepCopy()).Should(Succeed())

		deploymentObject = utils.ExampleDeployment.DeepCopy()

		m.Create(deploymentObject).Should(Succeed())

		ownerRef = utils.GetOwnerRefDeployment(deploymentObject)

		stopMgr, mgrStopped = StartTestManager(mgr)
		m.Get(deploymentObject, timeout).Should(Succeed())
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

	Context("When a legacy deployment is reconciled", func() {
		var cm1 *corev1.ConfigMap
		var cm2 *corev1.ConfigMap
		var s1 *corev1.Secret
		var s2 *corev1.Secret

		BeforeEach(func() {
			cm1 = utils.ExampleConfigMap1.DeepCopy()
			cm2 = utils.ExampleConfigMap2.DeepCopy()
			s1 = utils.ExampleSecret1.DeepCopy()
			s2 = utils.ExampleSecret2.DeepCopy()

			for _, obj := range []Object{cm1, cm2, s1, s2} {
				m.Update(obj, func(obj client.Object) client.Object {
					obj.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
					return obj
				}, timeout).Should(Succeed())
			}

			f := deploymentObject.GetFinalizers()
			f = append(f, FinalizerString)
			m.Update(deploymentObject, func(obj client.Object) client.Object {
				obj.SetFinalizers(f)
				return obj
			}, timeout).Should(Succeed())

			_, err := h.handlePodController(deploymentObject)
			Expect(err).NotTo(HaveOccurred())
		})

		It("removes owner references from all children", func() {
			for _, obj := range []Object{cm1, cm2, s1, s2} {
				m.Get(obj, timeout).Should(Succeed())
				Eventually(obj, timeout).ShouldNot(utils.WithOwnerReferences(ContainElement(ownerRef)))
			}
		})

		It("removes the finalizer from the deployment", func() {
			m.Get(deploymentObject, timeout).Should(Succeed())
			Eventually(deploymentObject, timeout).ShouldNot(utils.WithFinalizers(ContainElement(FinalizerString)))
		})
	})

})
