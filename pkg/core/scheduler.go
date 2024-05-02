package core

// disableScheduling sets an invalid scheduler and adds an annotation with the original scheduler
func disableScheduling(obj podController) {
	if isSchedulingDisabled(obj) {
		return
	}

	// Get the existing annotations
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Store previous scheduler in annotation
	schedulerName := obj.GetPodTemplate().Spec.SchedulerName
	annotations[SchedulingDisabledAnnotation] = schedulerName
	obj.SetAnnotations(annotations)

	// Set invalid scheduler
	podTemplate := obj.GetPodTemplate()
	podTemplate.Spec.SchedulerName = SchedulingDisabledSchedulerName
	obj.SetPodTemplate(podTemplate)
}

// isSchedulingDisabled returns true if scheduling has been disabled by wave
func isSchedulingDisabled(obj podController) bool {
	// Get the existing annotations
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return false
	}
	_, ok := annotations[SchedulingDisabledAnnotation]
	return ok
}

// enableScheduling restore scheduling if it has been disabled by wave
func restoreScheduling(obj podController) {
	// Get the existing annotations
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	schedulerName, ok := annotations[SchedulingDisabledAnnotation]
	if !ok {
		// Scheduling has not been disabled
		return
	}
	delete(annotations, SchedulingDisabledAnnotation)
	obj.SetAnnotations(annotations)

	// Restore scheduler
	podTemplate := obj.GetPodTemplate()
	podTemplate.Spec.SchedulerName = schedulerName
	obj.SetPodTemplate(podTemplate)
}
