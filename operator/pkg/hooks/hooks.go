package hooks

import (
	"context"
	"time"

	steerv1alpha1 "github.com/MrLYC/steer/operator/api/v1alpha1"
)

type Stage string

const (
	StagePreTest  Stage = "preTest"
	StagePostTest Stage = "postTest"
)

// Executor runs hooks defined on HelmTestJob.
//
// Block A note: this is an interface-only abstraction. The real implementation
// (script + kubernetes) will be added in later blocks.
type Executor interface {
	Execute(ctx context.Context, req ExecuteRequest) ([]Result, error)
}

type ExecuteRequest struct {
	Namespace string
	Stage     Stage
	Hooks     []steerv1alpha1.Hook
}

type Result struct {
	Name        string
	Stage       Stage
	Succeeded   bool
	Message     string
	StartedAt   *time.Time
	CompletedAt *time.Time
}

// FakeExecutor is a simple injectable fake implementation of Executor.
type FakeExecutor struct {
	ExecuteFunc func(ctx context.Context, req ExecuteRequest) ([]Result, error)
}

func (f *FakeExecutor) Execute(ctx context.Context, req ExecuteRequest) ([]Result, error) {
	if f.ExecuteFunc != nil {
		return f.ExecuteFunc(ctx, req)
	}
	return nil, nil
}
