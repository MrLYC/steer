package cleanup

import "context"

// Runner encapsulates post-test or post-release cleanup behavior.
//
// Block A note: the real implementation (namespace deletion, image cleanup)
// will be added later.
type Runner interface {
	CleanupNamespace(ctx context.Context, namespace string, opts Options) error
}

type Options struct {
	DeleteNamespace bool
	DeleteImages    bool
}

// FakeRunner is a simple injectable fake implementation of Runner.
type FakeRunner struct {
	CleanupNamespaceFunc func(ctx context.Context, namespace string, opts Options) error
}

func (f *FakeRunner) CleanupNamespace(ctx context.Context, namespace string, opts Options) error {
	if f.CleanupNamespaceFunc != nil {
		return f.CleanupNamespaceFunc(ctx, namespace, opts)
	}
	return nil
}
