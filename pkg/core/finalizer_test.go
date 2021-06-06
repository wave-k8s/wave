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
