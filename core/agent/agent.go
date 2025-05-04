import (
    "github.com/robfig/cron/v3"
    "github.com/mudler/LocalAGI/core/types"
)

type Agent struct {
    ID          string
    Cron        *cron.Cron
    Scheduler   cron.EntryID
    // ...existing fields
}

func (a *Agent) InitCron() {
    a.Cron = cron.New()
    a.Cron.AddFunc(a.PeriodicRuns, func() {
        a.RunPeriodicTasks()
    })
    a.Cron.Start()
}

func (a *Agent) ScheduleTask(expression string, task func()) {
    entryID, _ := a.Cron.AddFunc(expression, task)
    a.Scheduler = entryID
}