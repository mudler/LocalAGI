import (
	"github.com/robfig/cron/v3"
	"github.com/mudler/LocalAGI/core/types"
)

type Agent struct {
	ID          string
	Cron        *cron.Cron
	Scheduler   cron.EntryID
)

func (a *Agent) InitCron() {
	a.Cron = cron.New()
	// Load existing tasks from state
	a.Cron.AddFunc(a.PeriodicRuns, func() { 
		// Trigger agent's periodic actions 
		a.RunPeriodicTasks()
	})
	a.Cron.Start()
}

func (a *Agent) ScheduleTask(expression string, task func()) {
	entryID, _ := a.Cron.AddFunc(expression, task)
	// Save task to state
}
