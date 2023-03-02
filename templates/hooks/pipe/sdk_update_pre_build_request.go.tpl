if pipeInMutatingState(latest) {
	return latest, requeueWaitWhileUpdating
}

// hack (continued from delta.go): if there is only a difference in the current
// and desired state (expressed through non-existing Spec field CurrentState,
// continuously requeue so we don't block changes to the resources to recover
// from a FAILED state
if !delta.DifferentExcept("Spec.CurrentState") {
	return latest, requeueWaitWhileUpdating
}
