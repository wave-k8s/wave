package core

// disableScheduling sets an invalid scheduler and adds an annotation with the original scheduler
func disableScheduling[I InstanceType](obj I) {
	if isSchedulingDisabled(obj) {
		return
	}

	// Get the existing annotations
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Store previous scheduler in annotation
	schedulerName := GetPodTemplate(obj).Spec.SchedulerName
	annotations[SchedulingDisabledAnnotation] = schedulerName
	obj.SetAnnotations(annotations)

	// Set invalid scheduler
	podTemplate := GetPodTemplate(obj)
	podTemplate.Spec.SchedulerName = SchedulingDisabledSchedulerName
	SetPodTemplate(obj, podTemplate)
}

// isSchedulingDisabled returns true if scheduling has been disabled by wave
func isSchedulingDisabled[I InstanceType](obj I) bool {
	// Get the existing annotations
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return false
	}
	_, ok := annotations[SchedulingDisabledAnnotation]
	return ok
}

// enableScheduling restore scheduling if it has been disabled by wave
func restoreScheduling[I InstanceType](obj I) {
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
	podTemplate := GetPodTemplate(obj)
	podTemplate.Spec.SchedulerName = schedulerName
	SetPodTemplate(obj, podTemplate)
}
