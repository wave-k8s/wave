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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/wave-k8s/wave/test/utils"
	appsv1 "k8s.io/api/apps/v1"
)

var _ = Describe("Wave finalizer Suite", func() {
	var deploymentObject *appsv1.Deployment
	var podControllerDeployment podController

	BeforeEach(func() {
		deploymentObject = utils.ExampleDeployment.DeepCopy()
		podControllerDeployment = &deployment{deploymentObject}
	})

	Context("addFinalizer", func() {
		It("adds the wave finalizer to the deployment", func() {
			addFinalizer(podControllerDeployment)

			Expect(deploymentObject.GetFinalizers()).To(ContainElement(FinalizerString))
		})

		It("leaves existing finalizers in place", func() {
			f := deploymentObject.GetFinalizers()
			f = append(f, "kubernetes")
			deploymentObject.SetFinalizers(f)
			addFinalizer(podControllerDeployment)

			Expect(deploymentObject.GetFinalizers()).To(ContainElement("kubernetes"))
		})
	})

	Context("removeFinalizer", func() {
		It("removes the wave finalizer from the deployment", func() {
			f := deploymentObject.GetFinalizers()
			f = append(f, FinalizerString)
			deploymentObject.SetFinalizers(f)
			removeFinalizer(podControllerDeployment)

			Expect(deploymentObject.GetFinalizers()).NotTo(ContainElement(FinalizerString))
		})

		It("leaves existing finalizers in place", func() {
			f := deploymentObject.GetFinalizers()
			f = append(f, "kubernetes")
			deploymentObject.SetFinalizers(f)
			removeFinalizer(podControllerDeployment)

			Expect(deploymentObject.GetFinalizers()).To(ContainElement("kubernetes"))
		})
	})

	Context("hasFinalizer", func() {
		It("returns true if the deployment has the finalizer", func() {
			f := deploymentObject.GetFinalizers()
			f = append(f, FinalizerString)
			deploymentObject.SetFinalizers(f)

			Expect(hasFinalizer(podControllerDeployment)).To(BeTrue())
		})

		It("returns false if the deployment doesn't have the finalizer", func() {
			// Test without any finalizers
			Expect(hasFinalizer(podControllerDeployment)).To(BeFalse())

			// Test with a different finalizer
			f := deploymentObject.GetFinalizers()
			f = append(f, "kubernetes")
			deploymentObject.SetFinalizers(f)
			Expect(hasFinalizer(podControllerDeployment)).To(BeFalse())
		})
	})
})
