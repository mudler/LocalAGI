package types

// SetResult sets the result of a job
func (j *JobResult) SetResult(text ActionState) {
	j.Lock()
	defer j.Unlock()

	j.State = append(j.State, text)
}

// SetResult sets the result of a job
func (j *JobResult) Finish(e error) {
	j.Lock()
	defer j.Unlock()

	j.Error = e
	close(j.ready)
}

// SetResult sets the result of a job
func (j *JobResult) SetResponse(response string) {
	j.Lock()
	defer j.Unlock()

	j.Response = response
}

// WaitResult waits for the result of a job
func (j *JobResult) WaitResult() *JobResult {
	<-j.ready
	j.Lock()
	defer j.Unlock()
	return j
}
