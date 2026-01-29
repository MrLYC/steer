package helm

import (
	"context"

	steerv1alpha1 "github.com/MrLYC/steer/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Client is an abstraction over Helm operations.
//
// Block A note: this package intentionally does NOT depend on the Helm SDK yet.
// Controllers should depend on this interface for testability.
type Client interface {
	InstallOrUpgrade(ctx context.Context, req InstallOrUpgradeRequest) (ReleaseInfo, error)
	Uninstall(ctx context.Context, req UninstallRequest) error
	Test(ctx context.Context, req TestRequest) (TestResult, error)
}

type InstallOrUpgradeRequest struct {
	// ReleaseName is the Helm release name.
	ReleaseName string

	// Namespace is the target namespace for the Helm release.
	Namespace string

	Chart  steerv1alpha1.ChartSpec
	Values steerv1alpha1.ValuesSpec

	CreateNamespace bool
	Timeout         metav1.Duration
}

// UninstallRequest defines parameters to uninstall a Helm release.
type UninstallRequest struct {
	ReleaseName string
	Namespace   string
	Timeout     metav1.Duration
}

// TestRequest defines parameters to run helm test.
type TestRequest struct {
	ReleaseName string
	Namespace   string
	Timeout     metav1.Duration
	Filter      string
}

// ReleaseInfo is minimal information about an installed Helm release.
type ReleaseInfo struct {
	Name      string
	Namespace string
	Version   int64
	Status    string
}

// TestResult represents the output of a helm test run.
type TestResult struct {
	Succeeded bool
	Logs      []string
}

// FakeClient is a simple, injectable fake implementation of Client.
//
// It is intended for controller tests in envtest without pulling Helm SDK.
type FakeClient struct {
	InstallOrUpgradeFunc func(ctx context.Context, req InstallOrUpgradeRequest) (ReleaseInfo, error)
	UninstallFunc        func(ctx context.Context, req UninstallRequest) error
	TestFunc             func(ctx context.Context, req TestRequest) (TestResult, error)
}

func (f *FakeClient) InstallOrUpgrade(ctx context.Context, req InstallOrUpgradeRequest) (ReleaseInfo, error) {
	if f.InstallOrUpgradeFunc != nil {
		return f.InstallOrUpgradeFunc(ctx, req)
	}
	return ReleaseInfo{Name: req.ReleaseName, Namespace: req.Namespace}, nil
}

func (f *FakeClient) Uninstall(ctx context.Context, req UninstallRequest) error {
	if f.UninstallFunc != nil {
		return f.UninstallFunc(ctx, req)
	}
	return nil
}

func (f *FakeClient) Test(ctx context.Context, req TestRequest) (TestResult, error) {
	if f.TestFunc != nil {
		return f.TestFunc(ctx, req)
	}
	return TestResult{Succeeded: true}, nil
}
