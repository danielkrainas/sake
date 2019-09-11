package service

type ExpirationTriggerTask struct {
	Coordinator CoordinatorService
}

func (task *ExpirationTriggerTask) RunTask() error {
	return task.Coordinator.UpdateExpired()
}

type RecipeCleanupTask struct {
	Coordinator CoordinatorService
}

func (task *RecipeCleanupTask) RunTask() error {
	return task.Coordinator.ClearInactive()
}
