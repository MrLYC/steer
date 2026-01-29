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
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	steerv1alpha1 "github.com/MrLYC/steer/operator/api/v1alpha1"
)

// HelmTestJobReconciler reconciles a HelmTestJob object
type HelmTestJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=steer.steer.io,resources=helmtestjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=steer.steer.io,resources=helmtestjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=steer.steer.io,resources=helmtestjobs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HelmTestJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *HelmTestJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var job steerv1alpha1.HelmTestJob
	if err := r.Get(ctx, req.NamespacedName, &job); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	now := time.Now()

	if job.Status.Phase == "" {
		job.Status.Phase = steerv1alpha1.HelmTestJobPhasePending
	}

	res, next, err := computeNextScheduleTime(now, job.Spec.Schedule)
	if err != nil {
		logger.Error(err, "failed to compute next schedule time")
		job.Status.Phase = steerv1alpha1.HelmTestJobPhaseFailed
		job.Status.Message = err.Error()
		_ = r.Status().Update(ctx, &job)
		return ctrl.Result{}, err
	}

	job.Status.NextScheduleTime = &metav1.Time{Time: next}
	job.Status.Message = ""
	if err := r.Status().Update(ctx, &job); err != nil {
		return ctrl.Result{}, err
	}

	return res, nil
}

func computeNextScheduleTime(now time.Time, spec steerv1alpha1.ScheduleSpec) (ctrl.Result, time.Time, error) {
	switch spec.Type {
	case steerv1alpha1.ScheduleTypeOnce:
		next := now.Add(spec.Delay.Duration)
		return ctrl.Result{RequeueAfter: spec.Delay.Duration}, next, nil
	case steerv1alpha1.ScheduleTypeCron:
		// Block C note: we intentionally avoid bringing in a cron parsing dependency yet.
		// For now we support a minimal, non-empty cron string and schedule a short requeue.
		if spec.Cron == "" {
			return ctrl.Result{}, time.Time{}, fmt.Errorf("schedule.cron is required when type=cron")
		}
		loc, err := time.LoadLocation(spec.Timezone)
		if err != nil {
			return ctrl.Result{}, time.Time{}, fmt.Errorf("invalid schedule.timezone %q: %w", spec.Timezone, err)
		}
		next := now.In(loc).Add(time.Minute)
		return ctrl.Result{RequeueAfter: time.Minute}, next, nil
	default:
		return ctrl.Result{}, time.Time{}, fmt.Errorf("unsupported schedule.type %q", spec.Type)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelmTestJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&steerv1alpha1.HelmTestJob{}).
		Complete(r)
}
