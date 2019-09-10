package service

type ExpirationTriggerTask struct {
	Coordinator CoordinatorService
}

func (task *ExpirationTriggerTask) RunTask() error {
	return task.Coordinator.UpdateExpired()
}

type WorkflowCleanupTask struct {
	Coordinator CoordinatorService
}

func (task *WorkflowCleanupTask) RunTask() error {
	return task.Coordinator.ClearInactive()
}
