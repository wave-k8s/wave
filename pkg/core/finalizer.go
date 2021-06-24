package core

// addFinalizer adds the wave finalizer to the given PodController
func addFinalizer(obj podController) {
	finalizers := obj.GetFinalizers()
	for _, finalizer := range finalizers {
		if finalizer == FinalizerString {
			// podController already contains the finalizer
			return
		}
	}

	//podController doesn't contain the finalizer, so add it
	finalizers = append(finalizers, FinalizerString)
	obj.SetFinalizers(finalizers)
}

// removeFinalizer removes the wave finalizer from the given podController
func removeFinalizer(obj podController) {
	finalizers := obj.GetFinalizers()

	// Filter existing finalizers removing any that match the finalizerString
	newFinalizers := []string{}
	for _, finalizer := range finalizers {
		if finalizer != FinalizerString {
			newFinalizers = append(newFinalizers, finalizer)
		}
	}

	// Update the object's finalizers
	obj.SetFinalizers(newFinalizers)
}

// hasFinalizer checks for the presence of the Wave finalizer
func hasFinalizer(obj podController) bool {
	finalizers := obj.GetFinalizers()
	for _, finalizer := range finalizers {
		if finalizer == FinalizerString {
			// podController already contains the finalizer
			return true
		}
	}

	return false
}
