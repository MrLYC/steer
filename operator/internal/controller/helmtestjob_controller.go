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
	"os"
	"time"

	"github.com/robfig/cron/v3"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

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
	nowMeta := metav1.Now()

	if job.Status.Phase == "" {
		job.Status.Phase = steerv1alpha1.HelmTestJobPhasePending
	}

	res, next, err := computeNextScheduleTime(now, job.CreationTimestamp.Time, job.Spec.Schedule, job.Status.NextScheduleTime, job.Status.LastScheduleTime)
	if err != nil {
		logger.Error(err, "failed to compute next schedule time")
		job.Status.Phase = steerv1alpha1.HelmTestJobPhaseFailed
		job.Status.Message = err.Error()
		_ = r.Status().Update(ctx, &job)
		return ctrl.Result{}, err
	}
	// NextScheduleTime is cleared when a cron run is started.
	if job.Status.NextScheduleTime == nil {
		job.Status.NextScheduleTime = &metav1.Time{Time: next}
	}
	if job.Spec.Schedule.Type == steerv1alpha1.ScheduleTypeCron {
		job.Status.NextScheduleTime = &metav1.Time{Time: next}
	}

	// Determine run key and if we should execute now.
	shouldRun := false
	runKey := "once"

	switch job.Spec.Schedule.Type {
	case steerv1alpha1.ScheduleTypeOnce:
		// Run at (creationTimestamp + delay). Don't rerun after terminal.
		if job.Status.Phase != steerv1alpha1.HelmTestJobPhaseSucceeded && job.Status.Phase != steerv1alpha1.HelmTestJobPhaseFailed {
			if job.Status.NextScheduleTime != nil && !now.Before(job.Status.NextScheduleTime.Time) {
				shouldRun = true
			}
		}
	case steerv1alpha1.ScheduleTypeCron:
		// While running, keep executing current run.
		if job.Status.Phase == steerv1alpha1.HelmTestJobPhaseRunning {
			shouldRun = true
			if job.Status.LastScheduleTime != nil {
				runKey = fmt.Sprintf("r%d", job.Status.LastScheduleTime.Time.Unix())
			}
			break
		}

		// Start a new run when due.
		if job.Status.NextScheduleTime != nil && !now.Before(job.Status.NextScheduleTime.Time) {
			due := job.Status.NextScheduleTime.DeepCopy()
			job.Status.LastScheduleTime = due
			runKey = fmt.Sprintf("r%d", due.Time.Unix())
			job.Status.NextScheduleTime = nil
			job.Status.Phase = steerv1alpha1.HelmTestJobPhasePending
			job.Status.StartTime = nil
			job.Status.CompletionTime = nil
			job.Status.CurrentStage = ""
			job.Status.CurrentIndex = 0
			shouldRun = true
		}
	}

	if !shouldRun {
		job.Status.Message = ""
		if err := r.Status().Update(ctx, &job); err != nil {
			return ctrl.Result{}, err
		}
		return res, nil
	}

	if job.Status.Phase != steerv1alpha1.HelmTestJobPhaseRunning {
		job.Status.StartTime = &nowMeta
		job.Status.Phase = steerv1alpha1.HelmTestJobPhaseRunning
		job.Status.CompletionTime = nil
		job.Status.Message = ""
	}

	// Initialize stage for new runs.
	if job.Status.CurrentStage == "" {
		job.Status.CurrentStage = steerv1alpha1.HelmTestJobStagePreTest
		job.Status.CurrentIndex = 0
	}

	// Resolve image for all Jobs.
	image := job.Spec.Test.Image
	if image == "" {
		image = os.Getenv("STEER_JOB_IMAGE")
	}
	if image == "" {
		err := fmt.Errorf("missing test image: set spec.test.image or env STEER_JOB_IMAGE")
		job.Status.Phase = steerv1alpha1.HelmTestJobPhaseFailed
		job.Status.Message = err.Error()
		job.Status.CompletionTime = &nowMeta
		_ = r.Status().Update(ctx, &job)
		return ctrl.Result{}, err
	}

	// State machine: execute one stage/hook at a time.
	// We allow a few fast transitions (e.g., no hooks) in a single reconcile.
	for step := 0; step < 4; step++ {
		switch job.Status.CurrentStage {
		case steerv1alpha1.HelmTestJobStagePreTest:
			if int(job.Status.CurrentIndex) >= len(job.Spec.Hooks.PreTest) {
				job.Status.CurrentStage = steerv1alpha1.HelmTestJobStageTest
				job.Status.CurrentIndex = 0
				continue
			}
			h := job.Spec.Hooks.PreTest[job.Status.CurrentIndex]
			name := jobNameForHook(job.Name, runKey, "pre", int(job.Status.CurrentIndex))
			phase, msg, err := r.ensureHookJob(ctx, &job, name, image, h)
			if err != nil {
				job.Status.Phase = steerv1alpha1.HelmTestJobPhaseFailed
				job.Status.Message = err.Error()
				job.Status.CompletionTime = &nowMeta
				_ = r.Status().Update(ctx, &job)
				return ctrl.Result{}, err
			}
			if phase == steerv1alpha1.HelmTestJobPhaseSucceeded {
				job.Status.CurrentIndex++
				continue
			}
			if phase == steerv1alpha1.HelmTestJobPhaseFailed {
				job.Status.Phase = steerv1alpha1.HelmTestJobPhaseFailed
				job.Status.Message = msg
				job.Status.CompletionTime = &nowMeta
				_ = r.Status().Update(ctx, &job)
				return ctrl.Result{}, nil
			}
			job.Status.Message = msg
			_ = r.Status().Update(ctx, &job)
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil

		case steerv1alpha1.HelmTestJobStageTest:
			name := jobNameForTest(job.Name, runKey)
			phase, msg, err := r.ensureTestJob(ctx, &job, name, image)
			if err != nil {
				job.Status.Phase = steerv1alpha1.HelmTestJobPhaseFailed
				job.Status.Message = err.Error()
				job.Status.CompletionTime = &nowMeta
				_ = r.Status().Update(ctx, &job)
				return ctrl.Result{}, err
			}
			if phase == steerv1alpha1.HelmTestJobPhaseSucceeded {
				job.Status.CurrentStage = steerv1alpha1.HelmTestJobStagePostTest
				job.Status.CurrentIndex = 0
				continue
			}
			if phase == steerv1alpha1.HelmTestJobPhaseFailed {
				job.Status.Phase = steerv1alpha1.HelmTestJobPhaseFailed
				job.Status.Message = msg
				job.Status.CompletionTime = &nowMeta
				_ = r.Status().Update(ctx, &job)
				return ctrl.Result{}, nil
			}
			job.Status.Message = msg
			_ = r.Status().Update(ctx, &job)
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil

		case steerv1alpha1.HelmTestJobStagePostTest:
			if int(job.Status.CurrentIndex) >= len(job.Spec.Hooks.PostTest) {
				job.Status.Phase = steerv1alpha1.HelmTestJobPhaseSucceeded
				job.Status.CompletionTime = &nowMeta
				job.Status.Message = ""
				if err := r.Status().Update(ctx, &job); err != nil {
					return ctrl.Result{}, err
				}
				return res, nil
			}
			h := job.Spec.Hooks.PostTest[job.Status.CurrentIndex]
			name := jobNameForHook(job.Name, runKey, "post", int(job.Status.CurrentIndex))
			phase, msg, err := r.ensureHookJob(ctx, &job, name, image, h)
			if err != nil {
				job.Status.Phase = steerv1alpha1.HelmTestJobPhaseFailed
				job.Status.Message = err.Error()
				job.Status.CompletionTime = &nowMeta
				_ = r.Status().Update(ctx, &job)
				return ctrl.Result{}, err
			}
			if phase == steerv1alpha1.HelmTestJobPhaseSucceeded {
				job.Status.CurrentIndex++
				continue
			}
			if phase == steerv1alpha1.HelmTestJobPhaseFailed {
				job.Status.Phase = steerv1alpha1.HelmTestJobPhaseFailed
				job.Status.Message = msg
				job.Status.CompletionTime = &nowMeta
				_ = r.Status().Update(ctx, &job)
				return ctrl.Result{}, nil
			}
			job.Status.Message = msg
			_ = r.Status().Update(ctx, &job)
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil

		default:
			job.Status.CurrentStage = steerv1alpha1.HelmTestJobStagePreTest
			job.Status.CurrentIndex = 0
			continue
		}
	}

	// If we reached here, we made progress but didn't create a blocking Job.
	if err := r.Status().Update(ctx, &job); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: 0}, nil
}

func jobNameForHook(parentName, runKey, stage string, idx int) string {
	base := fmt.Sprintf("%s-%s-%s-%d", parentName, runKey, stage, idx)
	if len(validation.IsDNS1123Label(base)) == 0 && len(base) <= 63 {
		return base
	}
	// Fall back to truncation.
	if len(base) > 63 {
		base = base[:63]
		base = trimTrailingHyphen(base)
	}
	return base
}

func jobNameForTest(parentName, runKey string) string {
	base := fmt.Sprintf("%s-%s-test", parentName, runKey)
	if len(validation.IsDNS1123Label(base)) == 0 && len(base) <= 63 {
		return base
	}
	if len(base) > 63 {
		base = base[:63]
		base = trimTrailingHyphen(base)
	}
	return base
}

func trimTrailingHyphen(s string) string {
	for len(s) > 0 && s[len(s)-1] == '-' {
		s = s[:len(s)-1]
	}
	if s == "" {
		return "job"
	}
	return s
}

func (r *HelmTestJobReconciler) ensureHookJob(ctx context.Context, parent *steerv1alpha1.HelmTestJob, jobName, image string, hook steerv1alpha1.Hook) (steerv1alpha1.HelmTestJobPhase, string, error) {
	var kjob batchv1.Job
	key := types.NamespacedName{Name: jobName, Namespace: parent.Namespace}
	if err := r.Get(ctx, key, &kjob); err != nil {
		if !errors.IsNotFound(err) {
			return steerv1alpha1.HelmTestJobPhaseFailed, "", err
		}
		newJob := batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: jobName, Namespace: parent.Namespace}}
		if err := controllerutil.SetControllerReference(parent, &newJob, r.Scheme); err != nil {
			return steerv1alpha1.HelmTestJobPhaseFailed, "", err
		}

		container := corev1.Container{Name: "hook", Image: image, ImagePullPolicy: corev1.PullIfNotPresent}
		switch hook.Type {
		case steerv1alpha1.HookTypeScript:
			container.Command = []string{"/bin/sh", "-c", hook.Script}
		default:
			return steerv1alpha1.HelmTestJobPhaseFailed, "", fmt.Errorf("unsupported hook.type %q", hook.Type)
		}

		newJob.Spec.Template.Spec = corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers:    []corev1.Container{container},
		}
		newJob.Spec.BackoffLimit = ptrInt32(0)
		if err := r.Create(ctx, &newJob); err != nil {
			return steerv1alpha1.HelmTestJobPhaseFailed, "", err
		}
		return steerv1alpha1.HelmTestJobPhasePending, "hook job created", nil
	}

	return phaseFromJob(&kjob)
}

func (r *HelmTestJobReconciler) ensureTestJob(ctx context.Context, parent *steerv1alpha1.HelmTestJob, jobName, image string) (steerv1alpha1.HelmTestJobPhase, string, error) {
	var kjob batchv1.Job
	key := types.NamespacedName{Name: jobName, Namespace: parent.Namespace}
	if err := r.Get(ctx, key, &kjob); err != nil {
		if !errors.IsNotFound(err) {
			return steerv1alpha1.HelmTestJobPhaseFailed, "", err
		}
		newJob := batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: jobName, Namespace: parent.Namespace}}
		if err := controllerutil.SetControllerReference(parent, &newJob, r.Scheme); err != nil {
			return steerv1alpha1.HelmTestJobPhaseFailed, "", err
		}

		// Minimal placeholder command. Real helm execution can be wired later.
		container := corev1.Container{
			Name:            "test",
			Image:           image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/bin/sh", "-c", "echo helm test placeholder"},
		}
		newJob.Spec.Template.Spec = corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers:    []corev1.Container{container},
		}
		newJob.Spec.BackoffLimit = ptrInt32(0)
		if err := r.Create(ctx, &newJob); err != nil {
			return steerv1alpha1.HelmTestJobPhaseFailed, "", err
		}
		return steerv1alpha1.HelmTestJobPhasePending, "test job created", nil
	}

	return phaseFromJob(&kjob)
}

func phaseFromJob(job *batchv1.Job) (steerv1alpha1.HelmTestJobPhase, string, error) {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return steerv1alpha1.HelmTestJobPhaseFailed, "job failed", nil
		}
		if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
			return steerv1alpha1.HelmTestJobPhaseSucceeded, "job succeeded", nil
		}
	}
	if job.Status.Active > 0 {
		return steerv1alpha1.HelmTestJobPhaseRunning, "job running", nil
	}
	// If it exists but hasn't started yet.
	return steerv1alpha1.HelmTestJobPhasePending, "job pending", nil
}

func ptrInt32(v int32) *int32 { return &v }

func computeNextScheduleTime(now time.Time, creationTime time.Time, spec steerv1alpha1.ScheduleSpec, currentNext *metav1.Time, lastScheduleTime *metav1.Time) (ctrl.Result, time.Time, error) {
	switch spec.Type {
	case steerv1alpha1.ScheduleTypeOnce:
		next := creationTime.Add(spec.Delay.Duration)
		requeueAfter := next.Sub(now)
		if requeueAfter < 0 {
			requeueAfter = 0
		}
		return ctrl.Result{RequeueAfter: requeueAfter}, next, nil
	case steerv1alpha1.ScheduleTypeCron:
		if spec.Cron == "" {
			return ctrl.Result{}, time.Time{}, fmt.Errorf("schedule.cron is required when type=cron")
		}
		loc, err := time.LoadLocation(spec.Timezone)
		if err != nil {
			return ctrl.Result{}, time.Time{}, fmt.Errorf("invalid schedule.timezone %q: %w", spec.Timezone, err)
		}

		// Standard 5-field cron with descriptors.
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		schedule, err := parser.Parse(spec.Cron)
		if err != nil {
			return ctrl.Result{}, time.Time{}, fmt.Errorf("invalid schedule.cron %q: %w", spec.Cron, err)
		}

		// If we already have a next schedule time, keep it until it is due.
		if currentNext != nil {
			next := currentNext.Time
			requeueAfter := next.Sub(now)
			if requeueAfter < 0 {
				requeueAfter = 0
			}
			return ctrl.Result{RequeueAfter: requeueAfter}, next, nil
		}

		nowInLoc := now.In(loc)
		anchor := nowInLoc
		if lastScheduleTime != nil {
			// Avoid scheduling immediately after a run by anchoring to lastScheduleTime.
			anchor = lastScheduleTime.Time.In(loc)
		}
		next := schedule.Next(anchor)
		requeueAfter := next.Sub(nowInLoc)
		if requeueAfter < 0 {
			requeueAfter = 0
		}
		return ctrl.Result{RequeueAfter: requeueAfter}, next, nil
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
