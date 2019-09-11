package service

type ExpirationTrigger struct {
	Coordinator CoordinatorService
}

func (task *ExpirationTrigger) RunTask() error {
	return task.Coordinator.UpdateExpired()
}
