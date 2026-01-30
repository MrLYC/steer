/*
Copyright 2026 MrLYC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HelmReleaseRef references a HelmRelease resource.
type HelmReleaseRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// +kubebuilder:validation:Enum=once;cron
type ScheduleType string

const (
	ScheduleTypeOnce ScheduleType = "once"
	ScheduleTypeCron ScheduleType = "cron"
)

type ScheduleSpec struct {
	// Type decides schedule behavior.
	// +kubebuilder:validation:Required
	Type ScheduleType `json:"type"`

	// Delay is only meaningful for type=once.
	// +optional
	Delay metav1.Duration `json:"delay,omitempty"`

	// Cron is only meaningful for type=cron.
	// +optional
	Cron string `json:"cron,omitempty"`

	// Timezone is used for cron schedules.
	// +kubebuilder:default="Asia/Shanghai"
	// +optional
	Timezone string `json:"timezone,omitempty"`
}

type TestSpec struct {
	// Image is the container image used to run helm test.
	// If empty, the controller will fall back to env var STEER_JOB_IMAGE.
	// +optional
	Image string `json:"image,omitempty"`

	// Timeout is the helm test timeout.
	// +kubebuilder:default="10m"
	// +optional
	Timeout metav1.Duration `json:"timeout,omitempty"`

	// Logs controls whether to show logs.
	// +kubebuilder:default=true
	// +optional
	Logs *bool `json:"logs,omitempty"`

	// Filter limits the executed tests.
	// +optional
	Filter string `json:"filter,omitempty"`
}

// +kubebuilder:validation:Enum=script;kubernetes
type HookType string

const (
	HookTypeScript     HookType = "script"
	HookTypeKubernetes HookType = "kubernetes"
)

type HookEnvVarSource struct {
	// FieldPath references the HelmTestJob object.
	// Example: status.phase
	// +optional
	FieldPath string `json:"fieldPath,omitempty"`

	// HelmReleaseRef references the referenced HelmRelease object.
	// +optional
	HelmReleaseRef *HookEnvVarHelmReleaseRefSource `json:"helmReleaseRef,omitempty"`
}

type HookEnvVarHelmReleaseRefSource struct {
	// FieldPath references the HelmRelease object.
	// Example: spec.deployment.namespace
	// +kubebuilder:validation:Required
	FieldPath string `json:"fieldPath"`
}

type HookEnvVar struct {
	Name string `json:"name"`

	// Value is a literal value.
	// +optional
	Value string `json:"value,omitempty"`

	// ValueFrom references a field.
	// +optional
	ValueFrom *HookEnvVarSource `json:"valueFrom,omitempty"`
}

type KubernetesHookSpec struct {
	// Object is the embedded Kubernetes object (Job, Pod, etc).
	// +kubebuilder:validation:EmbeddedResource
	// +kubebuilder:pruning:PreserveUnknownFields
	k8sruntime.RawExtension `json:",inline"`
}

type Hook struct {
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	Type HookType `json:"type"`

	// Env injects environment variables for script/kubernetes hooks.
	// +optional
	Env []HookEnvVar `json:"env,omitempty"`

	// Script is only meaningful for type=script.
	// +optional
	Script string `json:"script,omitempty"`

	// Kubernetes is only meaningful for type=kubernetes.
	// +optional
	Kubernetes *KubernetesHookSpec `json:"kubernetes,omitempty"`
}

type HooksSpec struct {
	// PreTest hooks are executed before helm test.
	// +optional
	PreTest []Hook `json:"preTest,omitempty"`

	// PostTest hooks are executed after helm test.
	// +optional
	PostTest []Hook `json:"postTest,omitempty"`
}

type HelmTestJobCleanupSpec struct {
	// DeleteNamespace controls whether to delete the namespace.
	// +optional
	DeleteNamespace *bool `json:"deleteNamespace,omitempty"`

	// DeleteImages controls whether to delete images (if supported).
	// +optional
	DeleteImages *bool `json:"deleteImages,omitempty"`
}

// HelmTestJobSpec defines the desired state of HelmTestJob
type HelmTestJobSpec struct {
	// HelmReleaseRef points to an existing HelmRelease.
	// +kubebuilder:validation:Required
	HelmReleaseRef HelmReleaseRef `json:"helmReleaseRef"`

	// Schedule defines once/cron execution.
	// +kubebuilder:validation:Required
	Schedule ScheduleSpec `json:"schedule"`

	// Test config for helm test.
	// +optional
	Test TestSpec `json:"test,omitempty"`

	// Hooks to run before/after test.
	// +optional
	Hooks HooksSpec `json:"hooks,omitempty"`

	// Cleanup can override HelmRelease cleanup settings.
	// +optional
	Cleanup *HelmTestJobCleanupSpec `json:"cleanup,omitempty"`
}

// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed
type HelmTestJobPhase string

const (
	HelmTestJobPhasePending   HelmTestJobPhase = "Pending"
	HelmTestJobPhaseRunning   HelmTestJobPhase = "Running"
	HelmTestJobPhaseSucceeded HelmTestJobPhase = "Succeeded"
	HelmTestJobPhaseFailed    HelmTestJobPhase = "Failed"
)

type TestResult struct {
	Name string `json:"name"`

	// Phase is per-test result.
	// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed
	Phase HelmTestJobPhase `json:"phase"`

	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`
	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`
	// +optional
	Logs string `json:"logs,omitempty"`
}

type HookResult struct {
	Name string `json:"name"`
	// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed
	Phase HelmTestJobPhase `json:"phase"`
	// +optional
	Message string `json:"message,omitempty"`
}

type HookResults struct {
	// +optional
	PreTest []HookResult `json:"preTest,omitempty"`
	// +optional
	PostTest []HookResult `json:"postTest,omitempty"`
}

// HelmTestJobStatus defines the observed state of HelmTestJob
type HelmTestJobStatus struct {
	// Phase indicates current state.
	// +optional
	Phase HelmTestJobPhase `json:"phase,omitempty"`

	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// +optional
	TestResults []TestResult `json:"testResults,omitempty"`

	// +optional
	HookResults *HookResults `json:"hookResults,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`

	// NextScheduleTime is only meaningful for cron schedules.
	// +optional
	NextScheduleTime *metav1.Time `json:"nextScheduleTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Next",type=string,JSONPath=`.status.nextScheduleTime`,priority=1
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// HelmTestJob is the Schema for the helmtestjobs API
type HelmTestJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HelmTestJobSpec   `json:"spec,omitempty"`
	Status HelmTestJobStatus `json:"status,omitempty"`
}

func init() {
	SchemeBuilder.Register(&HelmTestJob{}, &HelmTestJobList{})
}

//+kubebuilder:object:root=true

// HelmTestJobList contains a list of HelmTestJob
type HelmTestJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HelmTestJob `json:"items"`
}
