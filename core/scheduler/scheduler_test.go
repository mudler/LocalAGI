package scheduler_test

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/mudler/LocalAGI/core/scheduler"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// MockExecutor for testing
type MockExecutor struct {
	executedTasks []string
	shouldError   bool
}

func (m *MockExecutor) Execute(ctx context.Context, agentName string, prompt string) (*scheduler.JobResult, error) {
	m.executedTasks = append(m.executedTasks, agentName+":"+prompt)
	if m.shouldError {
		return nil, errors.New("mock execution error")
	}
	return &scheduler.JobResult{Response: "test response"}, nil
}

var _ = Describe("Scheduler", func() {
	var (
		tempFile string
		store    scheduler.TaskStore
		executor *MockExecutor
		sched    *scheduler.Scheduler
	)

	BeforeEach(func() {
		// Create temporary file for JSON store
		f, err := os.CreateTemp("", "scheduler_test_*.json")
		Expect(err).NotTo(HaveOccurred())
		tempFile = f.Name()
		f.Close()

		store, err = scheduler.NewJSONStore(tempFile)
		Expect(err).NotTo(HaveOccurred())

		executor = &MockExecutor{}
		sched = scheduler.NewScheduler(store, executor, 100*time.Millisecond)
		sched.Start()
	})

	AfterEach(func() {
		if sched != nil {
			sched.Stop()
		}
		os.Remove(tempFile)
	})

	Describe("Task Creation", func() {
		It("should create a valid task with cron schedule", func() {
			task, err := scheduler.NewTask("test-agent", "test prompt", scheduler.ScheduleTypeCron, "0 0 * * *")
			Expect(err).NotTo(HaveOccurred())
			Expect(task.ID).NotTo(BeEmpty())
			Expect(task.AgentName).To(Equal("test-agent"))
			Expect(task.Prompt).To(Equal("test prompt"))
			Expect(task.ScheduleType).To(Equal(scheduler.ScheduleTypeCron))
			Expect(task.Status).To(Equal(scheduler.TaskStatusActive))
			Expect(task.NextRun).NotTo(BeZero())
		})

		It("should return error for invalid cron expression", func() {
			_, err := scheduler.NewTask("test-agent", "test prompt", scheduler.ScheduleTypeCron, "invalid cron")
			Expect(err).To(HaveOccurred())
		})

		It("should create a valid task with interval schedule", func() {
			task, err := scheduler.NewTask("test-agent", "test prompt", scheduler.ScheduleTypeInterval, "3600000")
			Expect(err).NotTo(HaveOccurred())
			Expect(task.NextRun).To(BeTemporally("~", time.Now().Add(time.Hour), 5*time.Second))
		})

		It("should create a valid task with once schedule", func() {
			futureTime := time.Now().Add(24 * time.Hour)
			task, err := scheduler.NewTask("test-agent", "test prompt", scheduler.ScheduleTypeOnce, futureTime.Format(time.RFC3339))
			Expect(err).NotTo(HaveOccurred())
			Expect(task.NextRun).To(BeTemporally("~", futureTime, time.Second))
		})
	})

	Describe("Task IsDue", func() {
		It("should return true for active task past due time", func() {
			task := &scheduler.Task{
				Status:  scheduler.TaskStatusActive,
				NextRun: time.Now().Add(-1 * time.Hour),
			}
			Expect(task.IsDue()).To(BeTrue())
		})

		It("should return false for active task not yet due", func() {
			task := &scheduler.Task{
				Status:  scheduler.TaskStatusActive,
				NextRun: time.Now().Add(1 * time.Hour),
			}
			Expect(task.IsDue()).To(BeFalse())
		})

		It("should return false for paused task even if past due", func() {
			task := &scheduler.Task{
				Status:  scheduler.TaskStatusPaused,
				NextRun: time.Now().Add(-1 * time.Hour),
			}
			Expect(task.IsDue()).To(BeFalse())
		})
	})

	Describe("JSON Store", func() {
		Context("CRUD operations", func() {
			It("should create and retrieve a task", func() {
				task, err := scheduler.NewTask("test-agent", "test prompt", scheduler.ScheduleTypeCron, "0 0 * * *")
				Expect(err).NotTo(HaveOccurred())

				err = store.Create(task)
				Expect(err).NotTo(HaveOccurred())

				retrieved, err := store.Get(task.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.ID).To(Equal(task.ID))
				Expect(retrieved.AgentName).To(Equal(task.AgentName))
				Expect(retrieved.Prompt).To(Equal(task.Prompt))
			})

			It("should update a task", func() {
				task, _ := scheduler.NewTask("test-agent", "test prompt", scheduler.ScheduleTypeCron, "0 0 * * *")
				store.Create(task)

				task.Prompt = "updated prompt"
				err := store.Update(task)
				Expect(err).NotTo(HaveOccurred())

				updated, err := store.Get(task.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Prompt).To(Equal("updated prompt"))
			})

			It("should delete a task", func() {
				task, _ := scheduler.NewTask("test-agent", "test prompt", scheduler.ScheduleTypeCron, "0 0 * * *")
				store.Create(task)

				err := store.Delete(task.ID)
				Expect(err).NotTo(HaveOccurred())

				_, err = store.Get(task.ID)
				Expect(err).To(HaveOccurred())
			})

			It("should return error when getting non-existent task", func() {
				_, err := store.Get("non-existent-id")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("Querying tasks", func() {
			BeforeEach(func() {
				// Create test tasks
				task1, _ := scheduler.NewTask("agent1", "prompt1", scheduler.ScheduleTypeOnce, time.Now().Add(-1*time.Hour).Format(time.RFC3339))
				task2, _ := scheduler.NewTask("agent2", "prompt2", scheduler.ScheduleTypeOnce, time.Now().Add(1*time.Hour).Format(time.RFC3339))
				task3, _ := scheduler.NewTask("agent1", "prompt3", scheduler.ScheduleTypeOnce, time.Now().Add(-1*time.Hour).Format(time.RFC3339))
				task3.Status = scheduler.TaskStatusPaused

				store.Create(task1)
				store.Create(task2)
				store.Create(task3)
			})

			It("should get all tasks", func() {
				tasks, err := store.GetAll()
				Expect(err).NotTo(HaveOccurred())
				Expect(tasks).To(HaveLen(3))
			})

			It("should get only due tasks", func() {
				dueTasks, err := store.GetDue()
				Expect(err).NotTo(HaveOccurred())
				Expect(dueTasks).To(HaveLen(1))
				Expect(dueTasks[0].AgentName).To(Equal("agent1"))
				Expect(dueTasks[0].Prompt).To(Equal("prompt1"))
			})

			It("should get tasks by agent", func() {
				agentTasks, err := store.GetByAgent("agent1")
				Expect(err).NotTo(HaveOccurred())
				Expect(agentTasks).To(HaveLen(2))
			})
		})

		Context("Task runs", func() {
			It("should log and retrieve task runs", func() {
				task, _ := scheduler.NewTask("test-agent", "test prompt", scheduler.ScheduleTypeCron, "0 0 * * *")
				store.Create(task)

				run := scheduler.NewTaskRun(task.ID)
				run.Status = "success"
				run.Result = "test result"
				run.DurationMs = 1000

				err := store.LogRun(run)
				Expect(err).NotTo(HaveOccurred())

				runs, err := store.GetRuns(task.ID, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(runs).To(HaveLen(1))
				Expect(runs[0].Status).To(Equal("success"))
				Expect(runs[0].Result).To(Equal("test result"))
			})

			It("should limit returned runs", func() {
				task, _ := scheduler.NewTask("test-agent", "test prompt", scheduler.ScheduleTypeCron, "0 0 * * *")
				store.Create(task)

				// Create 5 runs
				for i := 0; i < 5; i++ {
					run := scheduler.NewTaskRun(task.ID)
					store.LogRun(run)
				}

				runs, err := store.GetRuns(task.ID, 3)
				Expect(err).NotTo(HaveOccurred())
				Expect(runs).To(HaveLen(3))
			})
		})

		Context("Persistence", func() {
			It("should persist data across store instances", func() {
				task, _ := scheduler.NewTask("test-agent", "test prompt", scheduler.ScheduleTypeCron, "0 0 * * *")
				store.Create(task)
				store.Close()

				// Create new store instance with same file
				newStore, err := scheduler.NewJSONStore(tempFile)
				Expect(err).NotTo(HaveOccurred())
				defer newStore.Close()

				retrieved, err := newStore.Get(task.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.ID).To(Equal(task.ID))
			})
		})
	})

	Describe("Scheduler Execution", func() {
		It("should execute a due task", func() {
			task, _ := scheduler.NewTask("test-agent", "test prompt", scheduler.ScheduleTypeOnce, time.Now().Add(-1*time.Second).Format(time.RFC3339))
			err := sched.CreateTask(task)
			Expect(err).NotTo(HaveOccurred())

			sched.Start()

			Eventually(func() int {
				return len(executor.executedTasks)
			}, "2s", "100ms").Should(Equal(1))

			Expect(executor.executedTasks[0]).To(Equal("test-agent:test prompt"))

			// Verify task run was logged
			runs, err := sched.GetTaskRuns(task.ID, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(runs).To(HaveLen(1))
			Expect(runs[0].Status).To(Equal("success"))

			// Verify one-time task was deleted
			_, err = sched.GetTask(task.ID)
			Expect(err).To(HaveOccurred())
		})

		It("should execute recurring tasks multiple times", func() {
			task, _ := scheduler.NewTask("test-agent", "recurring", scheduler.ScheduleTypeInterval, "500")
			task.NextRun = time.Now().Add(-1 * time.Second)
			err := sched.CreateTask(task)
			Expect(err).NotTo(HaveOccurred())

			sched.Start()

			Eventually(func() int {
				return len(executor.executedTasks)
			}, "3s", "100ms").Should(BeNumerically(">=", 2))

			// Verify task is still active
			updatedTask, err := sched.GetTask(task.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTask.Status).To(Equal(scheduler.TaskStatusActive))
		})

		It("should handle task execution errors", func() {
			executor.shouldError = true
			task, _ := scheduler.NewTask("test-agent", "error task", scheduler.ScheduleTypeOnce, time.Now().Add(-1*time.Second).Format(time.RFC3339))
			sched.CreateTask(task)

			sched.Start()

			Eventually(func() int {
				runs, _ := sched.GetTaskRuns(task.ID, 10)
				return len(runs)
			}, "2s", "100ms").Should(Equal(1))

			runs, _ := sched.GetTaskRuns(task.ID, 10)
			Expect(runs[0].Status).To(Equal("error"))
			Expect(runs[0].Error).NotTo(BeEmpty())
		})

		It("should not execute paused tasks", func() {
			task, _ := scheduler.NewTask("test-agent", "paused", scheduler.ScheduleTypeOnce, time.Now().Add(-1*time.Second).Format(time.RFC3339))
			task.Status = scheduler.TaskStatusPaused
			sched.CreateTask(task)

			sched.Start()

			Consistently(func() int {
				return len(executor.executedTasks)
			}, "1s", "100ms").Should(Equal(0))
		})
	})

	Describe("Task Management", func() {
		It("should pause and resume a task", func() {
			task, _ := scheduler.NewTask("test-agent", "test", scheduler.ScheduleTypeCron, "0 0 * * *")
			sched.CreateTask(task)

			err := sched.PauseTask(task.ID)
			Expect(err).NotTo(HaveOccurred())

			paused, _ := sched.GetTask(task.ID)
			Expect(paused.Status).To(Equal(scheduler.TaskStatusPaused))

			err = sched.ResumeTask(task.ID)
			Expect(err).NotTo(HaveOccurred())

			resumed, _ := sched.GetTask(task.ID)
			Expect(resumed.Status).To(Equal(scheduler.TaskStatusActive))
			Expect(resumed.NextRun).NotTo(BeZero())
		})

		It("should get tasks by agent", func() {
			task1, _ := scheduler.NewTask("agent1", "prompt1", scheduler.ScheduleTypeCron, "0 0 * * *")
			task2, _ := scheduler.NewTask("agent2", "prompt2", scheduler.ScheduleTypeCron, "0 0 * * *")
			task3, _ := scheduler.NewTask("agent1", "prompt3", scheduler.ScheduleTypeCron, "0 0 * * *")

			sched.CreateTask(task1)
			sched.CreateTask(task2)
			sched.CreateTask(task3)

			agent1Tasks, err := sched.GetTasksByAgent("agent1")
			Expect(err).NotTo(HaveOccurred())
			Expect(agent1Tasks).To(HaveLen(2))
		})

		It("should delete a task", func() {
			task, _ := scheduler.NewTask("test-agent", "test", scheduler.ScheduleTypeCron, "0 0 * * *")
			sched.CreateTask(task)

			err := sched.DeleteTask(task.ID)
			Expect(err).NotTo(HaveOccurred())

			_, err = sched.GetTask(task.ID)
			Expect(err).To(HaveOccurred())
		})
	})
})
