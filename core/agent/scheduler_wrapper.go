package agent

import (
	"github.com/mudler/LocalAGI/core/scheduler"
)

// schedulerWrapper wraps scheduler.Scheduler to implement types.TaskScheduler interface
type schedulerWrapper struct {
	*scheduler.Scheduler
}

func (w *schedulerWrapper) CreateTask(task interface{}) error {
	t, ok := task.(*scheduler.Task)
	if !ok {
		return nil
	}
	return w.Scheduler.CreateTask(t)
}

func (w *schedulerWrapper) GetAllTasks() ([]interface{}, error) {
	tasks, err := w.Scheduler.GetAllTasks()
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, len(tasks))
	for i, t := range tasks {
		result[i] = t
	}
	return result, nil
}

func (w *schedulerWrapper) GetTask(id string) (interface{}, error) {
	return w.Scheduler.GetTask(id)
}

func (w *schedulerWrapper) DeleteTask(id string) error {
	return w.Scheduler.DeleteTask(id)
}

func (w *schedulerWrapper) PauseTask(id string) error {
	return w.Scheduler.PauseTask(id)
}

func (w *schedulerWrapper) ResumeTask(id string) error {
	return w.Scheduler.ResumeTask(id)
}
