package types

import "time"

// JobResult represents the result of a job
// +k8s:deepcopy-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=jobr
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status`
// +kubebuilder:printcolumn:name="Created At",type=date,JSONPath=`.metadata.creationTimestamp`

type JobResult struct {
	// +kubebuilder:validation:Required
	ID string `json:"id"`

	// +kubebuilder:validation:Required
	Timestamp time.Time `json:"timestamp"`

	// Status represents the current state of the job
	// Possible values: Pending, Running, Succeeded, Failed, Canceled
	Status string `json:"status"`

	// +kubebuilder:validation:Required
	Result interface{} `json:"result"`

	// +kubebuilder:validation:Required
	Error string `json:"error"`
}

// JobResultList is a list of JobResult
// +k8s:deepcopy-gen=package
// +kubebuilder:object:root=true

type JobResultList struct {
	Items []JobResult `json:"items"`
}