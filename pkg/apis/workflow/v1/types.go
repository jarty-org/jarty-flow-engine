package v1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
const (
	WorkflowStatusPending = "Pending"
	WorkflowStatusRunning = "Running"
	WorkflowStatusSuccess = "Success"
	WorkflowStatusFailed  = "Failed"
)

type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              WorkflowSpec   `json:"spec"`
	Status            WorkflowStatus `json:"status"`
}

type WorkflowSpec struct {
	Entry string         `json:"entry"`
	Steps []WorkflowStep `json:"steps"`
}

type WorkflowStep struct {
	Name      string       `json:"name"`
	Contaienr v1.Container `json:"container"`
}

type WorkflowStatus struct {
	Status   string      `json:"status"`
	CreateAt metav1.Time `json:"createAt"`
	Message  string      `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkflowList is a list of Workflow resources
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Workflow `json:"items"`
}
