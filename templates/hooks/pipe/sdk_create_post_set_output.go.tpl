if !pipeAvailable(&resource{ko}) {
	return &resource{ko}, requeueWaitWhileCreating
}