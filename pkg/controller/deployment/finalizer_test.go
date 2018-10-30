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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pusher/wave/test/utils"
	appsv1 "k8s.io/api/apps/v1"
)

var _ = Describe("Wave finalizer Suite", func() {
	var deployment *appsv1.Deployment

	BeforeEach(func() {
		deployment = utils.ExampleDeployment.DeepCopy()
	})

	Context("addFinalizer", func() {
		It("adds the wave finalizer to the deployment", func() {
			addFinalizer(deployment)

			Expect(deployment.GetFinalizers()).To(ContainElement(finalizerString))
		})

		It("leaves existing finalizers in place", func() {
			f := deployment.GetFinalizers()
			f = append(f, "kubernetes")
			deployment.SetFinalizers(f)
			addFinalizer(deployment)

			Expect(deployment.GetFinalizers()).To(ContainElement("kubernetes"))
		})
	})

	Context("removeFinalizer", func() {
		It("removes the wave finalizer from the deployment", func() {
			f := deployment.GetFinalizers()
			f = append(f, finalizerString)
			deployment.SetFinalizers(f)
			removeFinalizer(deployment)

			Expect(deployment.GetFinalizers()).NotTo(ContainElement(finalizerString))
		})

		It("leaves existing finalizers in place", func() {
			f := deployment.GetFinalizers()
			f = append(f, "kubernetes")
			deployment.SetFinalizers(f)
			removeFinalizer(deployment)

			Expect(deployment.GetFinalizers()).To(ContainElement("kubernetes"))
		})
	})

	Context("hasFinalizer", func() {
		It("returns true if the deployment has the finalizer", func() {
			f := deployment.GetFinalizers()
			f = append(f, finalizerString)
			deployment.SetFinalizers(f)

			Expect(hasFinalizer(deployment)).To(BeTrue())
		})

		It("returns false if the deployment doesn't have the finalizer", func() {
			// Test without any finalizers
			Expect(hasFinalizer(deployment)).To(BeFalse())

			// Test with a different finalizer
			f := deployment.GetFinalizers()
			f = append(f, "kubernetes")
			deployment.SetFinalizers(f)
			Expect(hasFinalizer(deployment)).To(BeFalse())
		})
	})
})
