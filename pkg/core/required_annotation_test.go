package core

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/wave-k8s/wave/test/utils"
	appsv1 "k8s.io/api/apps/v1"
)

var _ = Describe("Wave required annotation Suite", func() {
	var deploymentObject *appsv1.Deployment
	var podControllerDeployment podController

	BeforeEach(func() {
		deploymentObject = utils.ExampleDeployment.DeepCopy()
		podControllerDeployment = &deployment{deploymentObject}
	})

	Context("hasRequiredAnnotation", func() {
		It("returns true when the annotation has value true", func() {
			annotations := deploymentObject.GetAnnotations()
			if annotations == nil {
				annotations = make(map[string]string)
			}
			annotations[RequiredAnnotation] = requiredAnnotationValue
			deploymentObject.SetAnnotations(annotations)

			Expect(hasRequiredAnnotation(podControllerDeployment)).To(BeTrue())
		})

		It("returns false when the annotation has value other than true", func() {
			annotations := deploymentObject.GetAnnotations()
			if annotations == nil {
				annotations = make(map[string]string)
			}
			annotations[RequiredAnnotation] = "false"
			deploymentObject.SetAnnotations(annotations)

			Expect(hasRequiredAnnotation(podControllerDeployment)).To(BeFalse())
		})

		It("returns false when the annotation is not set", func() {
			Expect(hasRequiredAnnotation(podControllerDeployment)).To(BeFalse())
		})

	})
})
