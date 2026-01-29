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

package controller

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	steerv1alpha1 "github.com/MrLYC/steer/operator/api/v1alpha1"
	"github.com/MrLYC/steer/operator/pkg/helm"
)

// HelmReleaseReconciler reconciles a HelmRelease object
type HelmReleaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Helm   helm.Client
}

//+kubebuilder:rbac:groups=steer.steer.io,resources=helmreleases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=steer.steer.io,resources=helmreleases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=steer.steer.io,resources=helmreleases/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HelmRelease object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *HelmReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var hr steerv1alpha1.HelmRelease
	if err := r.Get(ctx, req.NamespacedName, &hr); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if r.Helm == nil {
		logger.Info("helm client not configured")
		return ctrl.Result{}, nil
	}

	releaseName := hr.Name
	reqInstall := helm.InstallOrUpgradeRequest{
		ReleaseName:     releaseName,
		Namespace:       hr.Spec.Deployment.Namespace,
		Chart:           hr.Spec.Chart,
		Values:          hr.Spec.Values,
		CreateNamespace: hr.Spec.Deployment.CreateNamespace,
		Timeout:         hr.Spec.Deployment.Timeout,
	}

	info, err := r.Helm.InstallOrUpgrade(ctx, reqInstall)
	now := metav1.Now()
	if err != nil {
		hr.Status.Phase = steerv1alpha1.HelmReleasePhaseFailed
		hr.Status.Message = err.Error()
		_ = r.Status().Update(ctx, &hr)
		return ctrl.Result{}, err
	}

	hr.Status.Phase = steerv1alpha1.HelmReleasePhaseInstalled
	hr.Status.DeployedAt = &now
	hr.Status.Message = ""
	hr.Status.HelmRelease = &steerv1alpha1.HelmReleaseInfo{
		Name:    info.Name,
		Version: info.Version,
		Status:  info.Status,
	}
	if err := r.Status().Update(ctx, &hr); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelmReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&steerv1alpha1.HelmRelease{}).
		Complete(r)
}
