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
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ChartSource specifies where a Helm chart comes from.
// +kubebuilder:validation:Enum=repository;git;local
type ChartSource string

const (
	ChartSourceRepository ChartSource = "repository"
	ChartSourceGit        ChartSource = "git"
	ChartSourceLocal      ChartSource = "local"
)

// ChartSpec defines how to locate a Helm chart.
type ChartSpec struct {
	// Source indicates how the chart should be fetched.
	// +kubebuilder:default=repository
	Source ChartSource `json:"source,omitempty"`

	// Git specifies the Git repository chart source when source=git.
	Git *GitChartSpec `json:"git,omitempty"`

	// Repository specifies the Helm repository chart source when source=repository.
	Repository *RepositoryChartSpec `json:"repository,omitempty"`

	// Local specifies a local filesystem path chart source when source=local.
	Local *LocalChartSpec `json:"local,omitempty"`
}

type GitChartSpec struct {
	// URL is the Git repository URL.
	URL string `json:"url"`
	// Ref is the Git ref (branch, tag, or commit).
	Ref string `json:"ref,omitempty"`
	// Path is the path to the chart within the repository.
	Path string `json:"path"`
}

type RepositoryChartSpec struct {
	// URL is the Helm repository URL.
	URL string `json:"url"`
	// Name is the chart name.
	Name string `json:"name"`
	// Version is the chart version.
	Version string `json:"version,omitempty"`
}

type LocalChartSpec struct {
	// Path is the local filesystem path to a chart directory.
	Path string `json:"path"`
}

type ValuesSource struct {
	// ConfigMapKeyRef references a key within a ConfigMap.
	ConfigMapKeyRef *ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
	// SecretKeyRef references a key within a Secret.
	SecretKeyRef *SecretKeySelector `json:"secretKeyRef,omitempty"`
}

type ConfigMapKeySelector struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type SecretKeySelector struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type ValuesSpec struct {
	// Inline contains raw YAML values.
	Inline string `json:"inline,omitempty"`
	// ValuesFrom references ConfigMaps/Secrets containing values YAML.
	ValuesFrom []ValuesSource `json:"valuesFrom,omitempty"`
}

type DeploymentSpec struct {
	// Namespace is the target namespace for the Helm release.
	Namespace string `json:"namespace"`
	// CreateNamespace indicates whether the namespace should be created.
	CreateNamespace bool `json:"createNamespace,omitempty"`
	// Timeout is the Helm install/upgrade timeout.
	Timeout metav1.Duration `json:"timeout,omitempty"`
	// Retries is the number of retries on failure.
	Retries int32 `json:"retries,omitempty"`
	// WaitAfterDeploy is an additional wait after deploy.
	WaitAfterDeploy metav1.Duration `json:"waitAfterDeploy,omitempty"`
	// AutoUninstallAfter is the duration after which the release should be uninstalled.
	AutoUninstallAfter metav1.Duration `json:"autoUninstallAfter,omitempty"`
}

type CleanupSpec struct {
	// DeleteNamespace indicates whether the target namespace should be deleted.
	DeleteNamespace bool `json:"deleteNamespace,omitempty"`
	// DeleteImages indicates whether images should be deleted (optional).
	DeleteImages bool `json:"deleteImages,omitempty"`
}

// HelmReleaseSpec defines the desired state of HelmRelease.
type HelmReleaseSpec struct {
	Chart      ChartSpec      `json:"chart"`
	Values     ValuesSpec     `json:"values,omitempty"`
	Deployment DeploymentSpec `json:"deployment"`
	Cleanup    CleanupSpec    `json:"cleanup,omitempty"`
}

// HelmReleasePhase defines the lifecycle phase of a HelmRelease.
// +kubebuilder:validation:Enum=Pending;Installing;Installed;Failed;Uninstalling;Uninstalled
type HelmReleasePhase string

const (
	HelmReleasePhasePending      HelmReleasePhase = "Pending"
	HelmReleasePhaseInstalling   HelmReleasePhase = "Installing"
	HelmReleasePhaseInstalled    HelmReleasePhase = "Installed"
	HelmReleasePhaseFailed       HelmReleasePhase = "Failed"
	HelmReleasePhaseUninstalling HelmReleasePhase = "Uninstalling"
	HelmReleasePhaseUninstalled  HelmReleasePhase = "Uninstalled"
)

type HelmReleaseInfo struct {
	Name    string `json:"name,omitempty"`
	Version int64  `json:"version,omitempty"`
	Status  string `json:"status,omitempty"`
}

// HelmReleaseStatus defines the observed state of HelmRelease.
type HelmReleaseStatus struct {
	Phase       HelmReleasePhase `json:"phase,omitempty"`
	DeployedAt  *metav1.Time     `json:"deployedAt,omitempty"`
	UninstallAt *metav1.Time     `json:"uninstallAt,omitempty"`
	Message     string           `json:"message,omitempty"`
	RetryCount  int32            `json:"retryCount,omitempty"`
	HelmRelease *HelmReleaseInfo `json:"helmRelease,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Deployed",type=date,JSONPath=`.status.deployedAt`,priority=1
//+kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.message`,priority=1

// HelmRelease is the Schema for the helmreleases API
type HelmRelease struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HelmReleaseSpec   `json:"spec,omitempty"`
	Status HelmReleaseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HelmReleaseList contains a list of HelmRelease
type HelmReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HelmRelease `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HelmRelease{}, &HelmReleaseList{})
}
