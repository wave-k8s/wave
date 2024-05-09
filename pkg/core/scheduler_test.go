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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/wave-k8s/wave/test/utils"
	appsv1 "k8s.io/api/apps/v1"
)

var _ = Describe("Wave scheduler Suite", func() {
	var deploymentObject *appsv1.Deployment
	var podControllerDeployment *appsv1.Deployment

	BeforeEach(func() {
		deploymentObject = utils.ExampleDeployment.DeepCopy()
		podControllerDeployment = deploymentObject
	})

	Context("When scheduler is disabled", func() {
		BeforeEach(func() {
			disableScheduling(podControllerDeployment)
		})

		It("Sets the annotations and stores the previous scheduler", func() {
			annotations := podControllerDeployment.GetAnnotations()
			Expect(annotations[SchedulingDisabledAnnotation]).To(Equal("default-scheduler"))
		})

		It("Disables scheduling", func() {
			podTemplate := GetPodTemplate(podControllerDeployment)
			Expect(podTemplate.Spec.SchedulerName).To(Equal(SchedulingDisabledSchedulerName))
		})

		It("Is reports as disabled", func() {
			Expect(isSchedulingDisabled(podControllerDeployment)).To(BeTrue())
		})

		Context("And Is Restored", func() {
			BeforeEach(func() {
				restoreScheduling(podControllerDeployment)
			})

			It("Removes the annotations", func() {
				annotations := podControllerDeployment.GetAnnotations()
				Expect(annotations).NotTo(HaveKey(SchedulingDisabledAnnotation))
			})

			It("Restores the scheduler", func() {
				podTemplate := GetPodTemplate(podControllerDeployment)
				Expect(podTemplate.Spec.SchedulerName).To(Equal("default-scheduler"))
			})

			It("Is does not report as disabled", func() {
				Expect(isSchedulingDisabled(podControllerDeployment)).To(BeFalse())
			})

		})

	})
})
