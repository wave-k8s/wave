package core

// hasRequiredAnnotation returns true if the given PodController has the wave
// annotation present
func hasRequiredAnnotation(obj podController) bool {
	annotations := obj.GetAnnotations()
	if value, ok := annotations[RequiredAnnotation]; ok {
		if value == requiredAnnotationValue {
			return true
		}
	}
	return false
}
