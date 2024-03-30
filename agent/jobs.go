package agent

import "sync"

// Job is a request to the agent to do something
type Job struct {
	// The job is a request to the agent to do something
	// It can be a question, a command, or a request to do something
	// The agent will try to do it, and return a response
	Text   string
	Image  string // base64 encoded image
	Result *JobResult
}

// JobResult is the result of a job
type JobResult struct {
	sync.Mutex
	// The result of a job
	Data  []string
	ready chan bool
}

// NewJobResult creates a new job result
func NewJobResult() *JobResult {
	return &JobResult{
		ready: make(chan bool),
	}
}

// NewJob creates a new job
// It is a request to the agent to do something
// It has a JobResult to get the result asynchronously
// To wait for a Job result, use JobResult.WaitResult()
func NewJob(text, image string) *Job {
	return &Job{
		Text:   text,
		Image:  image,
		Result: NewJobResult(),
	}
}

// SetResult sets the result of a job
func (j *JobResult) SetResult(text string) {
	j.Lock()
	defer j.Unlock()

	j.Data = append(j.Data, text)
}

// SetResult sets the result of a job
func (j *JobResult) Finish() {
	j.Lock()
	defer j.Unlock()

	close(j.ready)
}

// WaitResult waits for the result of a job
func (j *JobResult) WaitResult() []string {
	<-j.ready
	j.Lock()
	defer j.Unlock()
	return j.Data
}
