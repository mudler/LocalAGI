package types

type JobFilter interface {
	Name() string
	Apply(job *Job) (bool, error)
	IsTrigger() bool
}

type JobFilters []JobFilter

type FilterResult struct {
	HasTriggers bool `json:"has_triggers"`
	TriggeredBy string `json:"triggered_by,omitempty"`
	FailedBy string `json:"failed_by,omitempty"`
}
