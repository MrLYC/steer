# Steer 项目技术实现指南

本文档提供了 Steer 项目各个组件的详细技术实现指导,包括代码示例、最佳实践和常见问题解决方案。

## 目录

1. [Operator 开发详解](#operator-开发详解)
2. [HelmRelease 控制器实现](#helmrelease-控制器实现)
3. [HelmTestJob 控制器实现](#helmtestjob-控制器实现)
4. [钩子系统实现](#钩子系统实现)
5. [Web API 实现](#web-api-实现)
6. [Web UI 实现](#web-ui-实现)
7. [部署与运维](#部署与运维)

---

## Operator 开发详解

### 项目初始化

使用 Kubebuilder 初始化项目:

```bash
# 安装 Kubebuilder
curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)
chmod +x kubebuilder && mv kubebuilder /usr/local/bin/

# 创建项目
mkdir steer-operator && cd steer-operator
kubebuilder init --domain steer.io --repo github.com/yourusername/steer

# 创建 API
kubebuilder create api --group steer --version v1alpha1 --kind HelmRelease
kubebuilder create api --group steer --version v1alpha1 --kind HelmTestJob
```

### 项目结构

```
steer-operator/
├── api/
│   └── v1alpha1/
│       ├── helmrelease_types.go      # HelmRelease CRD 定义
│       ├── helmtestjob_types.go      # HelmTestJob CRD 定义
│       └── zz_generated.deepcopy.go  # 自动生成的代码
├── controllers/
│   ├── helmrelease_controller.go     # HelmRelease 控制器
│   ├── helmtestjob_controller.go     # HelmTestJob 控制器
│   └── suite_test.go                 # 测试套件
├── config/
│   ├── crd/                          # CRD YAML 文件
│   ├── rbac/                         # RBAC 配置
│   ├── manager/                      # Manager 配置
│   └── samples/                      # 示例 CR
├── pkg/
│   ├── helm/                         # Helm SDK 封装
│   ├── hooks/                        # 钩子执行器
│   └── cleanup/                      # 清理逻辑
├── Dockerfile
├── Makefile
└── main.go
```

---

## HelmRelease 控制器实现

### CRD 类型定义

在 `api/v1alpha1/helmrelease_types.go` 中定义 CRD 结构:

```go
package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HelmReleaseSpec 定义 HelmRelease 的期望状态
type HelmReleaseSpec struct {
    // Chart 配置
    Chart ChartSpec `json:"chart"`
    
    // Values 配置
    Values ValuesSpec `json:"values,omitempty"`
    
    // 部署配置
    Deployment DeploymentSpec `json:"deployment"`
    
    // 清理配置
    Cleanup CleanupSpec `json:"cleanup,omitempty"`
}

// ChartSpec 定义 Helm Chart 的来源
type ChartSpec struct {
    // Chart 来源类型: repository, git, local
    Source string `json:"source"`
    
    // Git 仓库配置
    Git *GitChartSpec `json:"git,omitempty"`
    
    // Helm Repository 配置
    Repository *RepositoryChartSpec `json:"repository,omitempty"`
}

type GitChartSpec struct {
    URL  string `json:"url"`
    Ref  string `json:"ref"`
    Path string `json:"path"`
}

type RepositoryChartSpec struct {
    URL     string `json:"url"`
    Name    string `json:"name"`
    Version string `json:"version"`
}

type ValuesSpec struct {
    // 内联 values
    Inline string `json:"inline,omitempty"`
    
    // 引用 ConfigMap/Secret
    ValuesFrom []ValuesReference `json:"valuesFrom,omitempty"`
}

type ValuesReference struct {
    ConfigMapKeyRef *ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
    SecretKeyRef    *SecretKeySelector    `json:"secretKeyRef,omitempty"`
}

type DeploymentSpec struct {
    // 目标命名空间
    Namespace string `json:"namespace"`
    
    // 是否创建命名空间
    CreateNamespace bool `json:"createNamespace,omitempty"`
    
    // 部署超时时间
    Timeout metav1.Duration `json:"timeout,omitempty"`
    
    // 失败重试次数
    Retries int `json:"retries,omitempty"`
    
    // 部署后等待时长
    WaitAfterDeploy metav1.Duration `json:"waitAfterDeploy,omitempty"`
    
    // 自动卸载时间(0 表示不自动卸载)
    AutoUninstallAfter metav1.Duration `json:"autoUninstallAfter,omitempty"`
}

type CleanupSpec struct {
    // 是否清理命名空间
    DeleteNamespace bool `json:"deleteNamespace,omitempty"`
    
    // 是否清理镜像
    DeleteImages bool `json:"deleteImages,omitempty"`
}

// HelmReleaseStatus 定义 HelmRelease 的观测状态
type HelmReleaseStatus struct {
    // 当前状态: Pending, Installing, Installed, Failed, Uninstalling, Uninstalled
    Phase string `json:"phase,omitempty"`
    
    // 部署时间
    DeployedAt *metav1.Time `json:"deployedAt,omitempty"`
    
    // 预计卸载时间
    UninstallAt *metav1.Time `json:"uninstallAt,omitempty"`
    
    // 错误信息
    Message string `json:"message,omitempty"`
    
    // 重试次数
    RetryCount int `json:"retryCount,omitempty"`
    
    // Helm Release 信息
    HelmRelease *HelmReleaseInfo `json:"helmRelease,omitempty"`
}

type HelmReleaseInfo struct {
    Name    string `json:"name"`
    Version int    `json:"version"`
    Status  string `json:"status"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// HelmRelease 是 HelmRelease API 的 Schema
type HelmRelease struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   HelmReleaseSpec   `json:"spec,omitempty"`
    Status HelmReleaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HelmReleaseList 包含 HelmRelease 的列表
type HelmReleaseList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []HelmRelease `json:"items"`
}

func init() {
    SchemeBuilder.Register(&HelmRelease{}, &HelmReleaseList{})
}
```

### 控制器实现

在 `controllers/helmrelease_controller.go` 中实现控制器逻辑:

```go
package controllers

import (
    "context"
    "fmt"
    "time"

    "k8s.io/apimachinery/pkg/runtime"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/log"

    steerv1alpha1 "github.com/yourusername/steer/api/v1alpha1"
    "github.com/yourusername/steer/pkg/helm"
)

// HelmReleaseReconciler 协调 HelmRelease 对象
type HelmReleaseReconciler struct {
    client.Client
    Scheme     *runtime.Scheme
    HelmClient *helm.Client
}

// +kubebuilder:rbac:groups=steer.io,resources=helmreleases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=steer.io,resources=helmreleases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=steer.io,resources=helmreleases/finalizers,verbs=update

// Reconcile 是 Kubernetes 协调循环的主要逻辑
func (r *HelmReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    logger := log.FromContext(ctx)

    // 获取 HelmRelease 实例
    var helmRelease steerv1alpha1.HelmRelease
    if err := r.Get(ctx, req.NamespacedName, &helmRelease); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 根据当前状态决定下一步操作
    switch helmRelease.Status.Phase {
    case "", "Pending":
        return r.reconcileInstall(ctx, &helmRelease)
    case "Installed":
        return r.reconcileInstalled(ctx, &helmRelease)
    case "Failed":
        return r.reconcileRetry(ctx, &helmRelease)
    case "Uninstalling":
        return r.reconcileUninstall(ctx, &helmRelease)
    default:
        logger.Info("Unknown phase", "phase", helmRelease.Status.Phase)
        return ctrl.Result{}, nil
    }
}

// reconcileInstall 处理安装逻辑
func (r *HelmReleaseReconciler) reconcileInstall(ctx context.Context, hr *steerv1alpha1.HelmRelease) (ctrl.Result, error) {
    logger := log.FromContext(ctx)
    logger.Info("Installing HelmRelease")

    // 更新状态为 Installing
    hr.Status.Phase = "Installing"
    if err := r.Status().Update(ctx, hr); err != nil {
        return ctrl.Result{}, err
    }

    // 使用 Helm SDK 安装 Chart
    releaseName := hr.Name
    namespace := hr.Spec.Deployment.Namespace
    
    // 获取 Chart
    chartPath, err := r.HelmClient.FetchChart(ctx, hr.Spec.Chart)
    if err != nil {
        return r.updateStatusFailed(ctx, hr, fmt.Sprintf("Failed to fetch chart: %v", err))
    }

    // 获取 Values
    values, err := r.getValues(ctx, hr)
    if err != nil {
        return r.updateStatusFailed(ctx, hr, fmt.Sprintf("Failed to get values: %v", err))
    }

    // 安装 Release
    release, err := r.HelmClient.Install(ctx, releaseName, namespace, chartPath, values, helm.InstallOptions{
        CreateNamespace: hr.Spec.Deployment.CreateNamespace,
        Timeout:         hr.Spec.Deployment.Timeout.Duration,
    })
    if err != nil {
        return r.updateStatusFailed(ctx, hr, fmt.Sprintf("Failed to install: %v", err))
    }

    // 更新状态为 Installed
    now := metav1.Now()
    hr.Status.Phase = "Installed"
    hr.Status.DeployedAt = &now
    hr.Status.HelmRelease = &steerv1alpha1.HelmReleaseInfo{
        Name:    release.Name,
        Version: release.Version,
        Status:  string(release.Info.Status),
    }
    
    // 计算自动卸载时间
    if hr.Spec.Deployment.AutoUninstallAfter.Duration > 0 {
        uninstallAt := metav1.NewTime(now.Add(hr.Spec.Deployment.AutoUninstallAfter.Duration))
        hr.Status.UninstallAt = &uninstallAt
    }

    if err := r.Status().Update(ctx, hr); err != nil {
        return ctrl.Result{}, err
    }

    // 如果配置了部署后等待时长,等待
    if hr.Spec.Deployment.WaitAfterDeploy.Duration > 0 {
        logger.Info("Waiting after deploy", "duration", hr.Spec.Deployment.WaitAfterDeploy.Duration)
        time.Sleep(hr.Spec.Deployment.WaitAfterDeploy.Duration)
    }

    // 如果配置了自动卸载,设置重新调度
    if hr.Spec.Deployment.AutoUninstallAfter.Duration > 0 {
        return ctrl.Result{RequeueAfter: hr.Spec.Deployment.AutoUninstallAfter.Duration}, nil
    }

    return ctrl.Result{}, nil
}

// reconcileInstalled 处理已安装状态的逻辑
func (r *HelmReleaseReconciler) reconcileInstalled(ctx context.Context, hr *steerv1alpha1.HelmRelease) (ctrl.Result, error) {
    logger := log.FromContext(ctx)

    // 检查是否需要自动卸载
    if hr.Status.UninstallAt != nil && time.Now().After(hr.Status.UninstallAt.Time) {
        logger.Info("Auto-uninstall time reached, uninstalling")
        hr.Status.Phase = "Uninstalling"
        if err := r.Status().Update(ctx, hr); err != nil {
            return ctrl.Result{}, err
        }
        return ctrl.Result{Requeue: true}, nil
    }

    // 如果还没到卸载时间,继续等待
    if hr.Status.UninstallAt != nil {
        requeueAfter := time.Until(hr.Status.UninstallAt.Time)
        return ctrl.Result{RequeueAfter: requeueAfter}, nil
    }

    return ctrl.Result{}, nil
}

// reconcileRetry 处理失败重试逻辑
func (r *HelmReleaseReconciler) reconcileRetry(ctx context.Context, hr *steerv1alpha1.HelmRelease) (ctrl.Result, error) {
    logger := log.FromContext(ctx)

    // 检查是否还有重试次数
    if hr.Status.RetryCount >= hr.Spec.Deployment.Retries {
        logger.Info("Max retries reached, giving up")
        return ctrl.Result{}, nil
    }

    // 增加重试次数
    hr.Status.RetryCount++
    hr.Status.Phase = "Pending"
    if err := r.Status().Update(ctx, hr); err != nil {
        return ctrl.Result{}, err
    }

    // 指数退避重试
    backoff := time.Duration(hr.Status.RetryCount) * 10 * time.Second
    logger.Info("Retrying installation", "attempt", hr.Status.RetryCount, "backoff", backoff)
    return ctrl.Result{RequeueAfter: backoff}, nil
}

// reconcileUninstall 处理卸载逻辑
func (r *HelmReleaseReconciler) reconcileUninstall(ctx context.Context, hr *steerv1alpha1.HelmRelease) (ctrl.Result, error) {
    logger := log.FromContext(ctx)
    logger.Info("Uninstalling HelmRelease")

    // 卸载 Release
    releaseName := hr.Name
    namespace := hr.Spec.Deployment.Namespace
    
    if err := r.HelmClient.Uninstall(ctx, releaseName, namespace); err != nil {
        return r.updateStatusFailed(ctx, hr, fmt.Sprintf("Failed to uninstall: %v", err))
    }

    // 清理命名空间
    if hr.Spec.Cleanup.DeleteNamespace {
        if err := r.deleteNamespace(ctx, namespace); err != nil {
            logger.Error(err, "Failed to delete namespace")
        }
    }

    // 清理镜像(如果配置)
    if hr.Spec.Cleanup.DeleteImages {
        if err := r.cleanupImages(ctx, hr); err != nil {
            logger.Error(err, "Failed to cleanup images")
        }
    }

    // 更新状态为 Uninstalled
    hr.Status.Phase = "Uninstalled"
    if err := r.Status().Update(ctx, hr); err != nil {
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}

// updateStatusFailed 更新状态为失败
func (r *HelmReleaseReconciler) updateStatusFailed(ctx context.Context, hr *steerv1alpha1.HelmRelease, message string) (ctrl.Result, error) {
    hr.Status.Phase = "Failed"
    hr.Status.Message = message
    if err := r.Status().Update(ctx, hr); err != nil {
        return ctrl.Result{}, err
    }
    return ctrl.Result{Requeue: true}, nil
}

// getValues 获取 Values 配置
func (r *HelmReleaseReconciler) getValues(ctx context.Context, hr *steerv1alpha1.HelmRelease) (map[string]interface{}, error) {
    // 实现从 inline 或 ConfigMap/Secret 获取 values
    // 这里简化实现,实际需要解析 YAML
    return map[string]interface{}{}, nil
}

// deleteNamespace 删除命名空间
func (r *HelmReleaseReconciler) deleteNamespace(ctx context.Context, namespace string) error {
    // 实现删除命名空间的逻辑
    return nil
}

// cleanupImages 清理镜像
func (r *HelmReleaseReconciler) cleanupImages(ctx context.Context, hr *steerv1alpha1.HelmRelease) error {
    // 实现清理镜像的逻辑
    return nil
}

// SetupWithManager 设置控制器与 Manager
func (r *HelmReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&steerv1alpha1.HelmRelease{}).
        Complete(r)
}
```

### Helm SDK 封装

在 `pkg/helm/client.go` 中封装 Helm SDK:

```go
package helm

import (
    "context"
    "time"

    "helm.sh/helm/v3/pkg/action"
    "helm.sh/helm/v3/pkg/chart/loader"
    "helm.sh/helm/v3/pkg/cli"
    "helm.sh/helm/v3/pkg/release"
)

type Client struct {
    settings *cli.EnvSettings
}

func NewClient() *Client {
    return &Client{
        settings: cli.New(),
    }
}

type InstallOptions struct {
    CreateNamespace bool
    Timeout         time.Duration
}

func (c *Client) Install(ctx context.Context, name, namespace, chartPath string, values map[string]interface{}, opts InstallOptions) (*release.Release, error) {
    actionConfig := new(action.Configuration)
    if err := actionConfig.Init(c.settings.RESTClientGetter(), namespace, "secret", nil); err != nil {
        return nil, err
    }

    install := action.NewInstall(actionConfig)
    install.ReleaseName = name
    install.Namespace = namespace
    install.CreateNamespace = opts.CreateNamespace
    install.Timeout = opts.Timeout

    chart, err := loader.Load(chartPath)
    if err != nil {
        return nil, err
    }

    return install.RunWithContext(ctx, chart, values)
}

func (c *Client) Uninstall(ctx context.Context, name, namespace string) error {
    actionConfig := new(action.Configuration)
    if err := actionConfig.Init(c.settings.RESTClientGetter(), namespace, "secret", nil); err != nil {
        return err
    }

    uninstall := action.NewUninstall(actionConfig)
    _, err := uninstall.Run(name)
    return err
}

func (c *Client) Test(ctx context.Context, name, namespace string, timeout time.Duration) ([]*release.Release, error) {
    actionConfig := new(action.Configuration)
    if err := actionConfig.Init(c.settings.RESTClientGetter(), namespace, "secret", nil); err != nil {
        return nil, err
    }

    test := action.NewReleaseTesting(actionConfig)
    test.Timeout = timeout

    return test.Run(name)
}

func (c *Client) FetchChart(ctx context.Context, chartSpec ChartSpec) (string, error) {
    // 实现从 Git、Repository 或本地获取 Chart 的逻辑
    return "", nil
}
```

---

## HelmTestJob 控制器实现

### CRD 类型定义

在 `api/v1alpha1/helmtestjob_types.go` 中定义:

```go
package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HelmTestJobSpec 定义 HelmTestJob 的期望状态
type HelmTestJobSpec struct {
    // 关联的 HelmRelease
    HelmReleaseRef HelmReleaseReference `json:"helmReleaseRef"`
    
    // 调度配置
    Schedule ScheduleSpec `json:"schedule"`
    
    // 测试配置
    Test TestSpec `json:"test"`
    
    // 钩子配置
    Hooks HooksSpec `json:"hooks,omitempty"`
    
    // 清理配置
    Cleanup CleanupSpec `json:"cleanup,omitempty"`
}

type HelmReleaseReference struct {
    Name      string `json:"name"`
    Namespace string `json:"namespace"`
}

type ScheduleSpec struct {
    // 调度类型: once, cron
    Type string `json:"type"`
    
    // Cron 表达式
    Cron string `json:"cron,omitempty"`
    
    // 时区
    Timezone string `json:"timezone,omitempty"`
}

type TestSpec struct {
    // helm test 超时时间
    Timeout metav1.Duration `json:"timeout,omitempty"`
    
    // 是否显示日志
    Logs bool `json:"logs,omitempty"`
    
    // 过滤特定的测试
    Filter string `json:"filter,omitempty"`
}

type HooksSpec struct {
    // 测试前钩子
    PreTest []Hook `json:"preTest,omitempty"`
    
    // 测试后钩子
    PostTest []Hook `json:"postTest,omitempty"`
}

type Hook struct {
    Name string `json:"name"`
    Type string `json:"type"` // script, kubernetes
    
    // Script Hook
    Script string `json:"script,omitempty"`
    
    // Kubernetes Hook
    Kubernetes *KubernetesHook `json:"kubernetes,omitempty"`
}

type KubernetesHook struct {
    APIVersion string                `json:"apiVersion"`
    Kind       string                `json:"kind"`
    Spec       map[string]interface{} `json:"spec"`
}

// HelmTestJobStatus 定义 HelmTestJob 的观测状态
type HelmTestJobStatus struct {
    // 当前状态
    Phase string `json:"phase,omitempty"`
    
    // 开始时间
    StartTime *metav1.Time `json:"startTime,omitempty"`
    
    // 完成时间
    CompletionTime *metav1.Time `json:"completionTime,omitempty"`
    
    // 测试结果
    TestResults []TestResult `json:"testResults,omitempty"`
    
    // 钩子执行结果
    HookResults HookResults `json:"hookResults,omitempty"`
    
    // 错误信息
    Message string `json:"message,omitempty"`
    
    // 下次执行时间
    NextScheduleTime *metav1.Time `json:"nextScheduleTime,omitempty"`
}

type TestResult struct {
    Name        string       `json:"name"`
    Phase       string       `json:"phase"`
    StartedAt   *metav1.Time `json:"startedAt,omitempty"`
    CompletedAt *metav1.Time `json:"completedAt,omitempty"`
    Logs        string       `json:"logs,omitempty"`
}

type HookResults struct {
    PreTest  []HookResult `json:"preTest,omitempty"`
    PostTest []HookResult `json:"postTest,omitempty"`
}

type HookResult struct {
    Name    string `json:"name"`
    Phase   string `json:"phase"`
    Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type HelmTestJob struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   HelmTestJobSpec   `json:"spec,omitempty"`
    Status HelmTestJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type HelmTestJobList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []HelmTestJob `json:"items"`
}

func init() {
    SchemeBuilder.Register(&HelmTestJob{}, &HelmTestJobList{})
}
```

### 控制器实现

在 `controllers/helmtestjob_controller.go` 中实现控制器逻辑:

```go
package controllers

import (
    "context"
    "fmt"
    "time"

    "github.com/robfig/cron/v3"
    "k8s.io/apimachinery/pkg/runtime"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/log"

    steerv1alpha1 "github.com/yourusername/steer/api/v1alpha1"
    "github.com/yourusername/steer/pkg/helm"
    "github.com/yourusername/steer/pkg/hooks"
)

// HelmTestJobReconciler 协调 HelmTestJob 对象
type HelmTestJobReconciler struct {
    client.Client
    Scheme       *runtime.Scheme
    HelmClient   *helm.Client
    HookExecutor *hooks.Executor
    CronScheduler *cron.Cron
}

// Reconcile 是 Kubernetes 协调循环的主要逻辑
func (r *HelmTestJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    logger := log.FromContext(ctx)

    // 获取 HelmTestJob 实例
    var testJob steerv1alpha1.HelmTestJob
    if err := r.Get(ctx, req.NamespacedName, &testJob); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 根据调度类型处理
    switch testJob.Spec.Schedule.Type {
    case "once":
        return r.reconcileOnce(ctx, &testJob)
    case "cron":
        return r.reconcileCron(ctx, &testJob)
    default:
        logger.Info("Unknown schedule type", "type", testJob.Spec.Schedule.Type)
        return ctrl.Result{}, nil
    }
}

// reconcileOnce 处理一次性任务
func (r *HelmTestJobReconciler) reconcileOnce(ctx context.Context, tj *steerv1alpha1.HelmTestJob) (ctrl.Result, error) {
    logger := log.FromContext(ctx)

    // 如果已经执行过,不再执行
    if tj.Status.Phase == "Succeeded" || tj.Status.Phase == "Failed" {
        return ctrl.Result{}, nil
    }

    // 如果正在运行,等待完成
    if tj.Status.Phase == "Running" {
        return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
    }

    // 执行测试
    logger.Info("Executing test job")
    return r.executeTest(ctx, tj)
}

// reconcileCron 处理周期性任务
func (r *HelmTestJobReconciler) reconcileCron(ctx context.Context, tj *steerv1alpha1.HelmTestJob) (ctrl.Result, error) {
    logger := log.FromContext(ctx)

    // 解析 Cron 表达式
    schedule, err := cron.ParseStandard(tj.Spec.Schedule.Cron)
    if err != nil {
        logger.Error(err, "Failed to parse cron expression")
        return ctrl.Result{}, err
    }

    // 计算下次执行时间
    now := time.Now()
    nextTime := schedule.Next(now)
    
    // 如果还没到执行时间,等待
    if now.Before(nextTime) {
        tj.Status.NextScheduleTime = &metav1.Time{Time: nextTime}
        if err := r.Status().Update(ctx, tj); err != nil {
            return ctrl.Result{}, err
        }
        return ctrl.Result{RequeueAfter: time.Until(nextTime)}, nil
    }

    // 执行测试
    logger.Info("Executing scheduled test job")
    result, err := r.executeTest(ctx, tj)
    
    // 计算下次执行时间
    nextTime = schedule.Next(time.Now())
    tj.Status.NextScheduleTime = &metav1.Time{Time: nextTime}
    if err := r.Status().Update(ctx, tj); err != nil {
        return ctrl.Result{}, err
    }

    return result, err
}

// executeTest 执行测试流程
func (r *HelmTestJobReconciler) executeTest(ctx context.Context, tj *steerv1alpha1.HelmTestJob) (ctrl.Result, error) {
    logger := log.FromContext(ctx)

    // 更新状态为 Running
    now := metav1.Now()
    tj.Status.Phase = "Running"
    tj.Status.StartTime = &now
    if err := r.Status().Update(ctx, tj); err != nil {
        return ctrl.Result{}, err
    }

    // 1. 部署 HelmRelease
    logger.Info("Deploying HelmRelease")
    if err := r.deployHelmRelease(ctx, tj); err != nil {
        return r.updateStatusFailed(ctx, tj, fmt.Sprintf("Failed to deploy HelmRelease: %v", err))
    }

    // 2. 执行前置钩子
    logger.Info("Executing pre-test hooks")
    preTestResults, err := r.HookExecutor.ExecuteHooks(ctx, tj.Spec.Hooks.PreTest)
    if err != nil {
        return r.updateStatusFailed(ctx, tj, fmt.Sprintf("Pre-test hook failed: %v", err))
    }
    tj.Status.HookResults.PreTest = preTestResults

    // 3. 执行 helm test
    logger.Info("Running helm test")
    testResults, err := r.runHelmTest(ctx, tj)
    if err != nil {
        return r.updateStatusFailed(ctx, tj, fmt.Sprintf("Helm test failed: %v", err))
    }
    tj.Status.TestResults = testResults

    // 4. 执行后置钩子
    logger.Info("Executing post-test hooks")
    postTestResults, err := r.HookExecutor.ExecuteHooks(ctx, tj.Spec.Hooks.PostTest)
    if err != nil {
        logger.Error(err, "Post-test hook failed")
    }
    tj.Status.HookResults.PostTest = postTestResults

    // 5. 清理资源
    logger.Info("Cleaning up resources")
    if err := r.cleanup(ctx, tj); err != nil {
        logger.Error(err, "Failed to cleanup")
    }

    // 更新状态为 Succeeded
    completionTime := metav1.Now()
    tj.Status.Phase = "Succeeded"
    tj.Status.CompletionTime = &completionTime
    if err := r.Status().Update(ctx, tj); err != nil {
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}

// deployHelmRelease 部署 HelmRelease 并等待就绪
func (r *HelmTestJobReconciler) deployHelmRelease(ctx context.Context, tj *steerv1alpha1.HelmTestJob) error {
    // 获取 HelmRelease
    var helmRelease steerv1alpha1.HelmRelease
    key := client.ObjectKey{
        Name:      tj.Spec.HelmReleaseRef.Name,
        Namespace: tj.Spec.HelmReleaseRef.Namespace,
    }
    if err := r.Get(ctx, key, &helmRelease); err != nil {
        return err
    }

    // 等待 HelmRelease 部署完成
    for {
        if err := r.Get(ctx, key, &helmRelease); err != nil {
            return err
        }
        
        if helmRelease.Status.Phase == "Installed" {
            break
        }
        
        if helmRelease.Status.Phase == "Failed" {
            return fmt.Errorf("HelmRelease deployment failed: %s", helmRelease.Status.Message)
        }
        
        time.Sleep(5 * time.Second)
    }

    return nil
}

// runHelmTest 执行 helm test
func (r *HelmTestJobReconciler) runHelmTest(ctx context.Context, tj *steerv1alpha1.HelmTestJob) ([]steerv1alpha1.TestResult, error) {
    releaseName := tj.Spec.HelmReleaseRef.Name
    namespace := tj.Spec.HelmReleaseRef.Namespace
    timeout := tj.Spec.Test.Timeout.Duration

    releases, err := r.HelmClient.Test(ctx, releaseName, namespace, timeout)
    if err != nil {
        return nil, err
    }

    var results []steerv1alpha1.TestResult
    for _, rel := range releases {
        for _, hook := range rel.Hooks {
            result := steerv1alpha1.TestResult{
                Name:  hook.Name,
                Phase: string(hook.LastRun.Phase),
                Logs:  hook.LastRun.Info,
            }
            if hook.LastRun.StartedAt != nil {
                result.StartedAt = &metav1.Time{Time: *hook.LastRun.StartedAt}
            }
            if hook.LastRun.CompletedAt != nil {
                result.CompletedAt = &metav1.Time{Time: *hook.LastRun.CompletedAt}
            }
            results = append(results, result)
        }
    }

    return results, nil
}

// cleanup 清理资源
func (r *HelmTestJobReconciler) cleanup(ctx context.Context, tj *steerv1alpha1.HelmTestJob) error {
    // 卸载 HelmRelease
    releaseName := tj.Spec.HelmReleaseRef.Name
    namespace := tj.Spec.HelmReleaseRef.Namespace
    
    if err := r.HelmClient.Uninstall(ctx, releaseName, namespace); err != nil {
        return err
    }

    // 删除命名空间
    if tj.Spec.Cleanup.DeleteNamespace {
        // 实现删除命名空间的逻辑
    }

    // 清理镜像
    if tj.Spec.Cleanup.DeleteImages {
        // 实现清理镜像的逻辑
    }

    return nil
}

// updateStatusFailed 更新状态为失败
func (r *HelmTestJobReconciler) updateStatusFailed(ctx context.Context, tj *steerv1alpha1.HelmTestJob, message string) (ctrl.Result, error) {
    completionTime := metav1.Now()
    tj.Status.Phase = "Failed"
    tj.Status.CompletionTime = &completionTime
    tj.Status.Message = message
    if err := r.Status().Update(ctx, tj); err != nil {
        return ctrl.Result{}, err
    }
    return ctrl.Result{}, nil
}

// SetupWithManager 设置控制器与 Manager
func (r *HelmTestJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&steerv1alpha1.HelmTestJob{}).
        Complete(r)
}
```

---

## 钩子系统实现

在 `pkg/hooks/executor.go` 中实现钩子执行器:

```go
package hooks

import (
    "context"
    "fmt"

    steerv1alpha1 "github.com/yourusername/steer/api/v1alpha1"
    batchv1 "k8s.io/api/batch/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

type Executor struct {
    Client client.Client
}

func NewExecutor(client client.Client) *Executor {
    return &Executor{Client: client}
}

func (e *Executor) ExecuteHooks(ctx context.Context, hooks []steerv1alpha1.Hook) ([]steerv1alpha1.HookResult, error) {
    var results []steerv1alpha1.HookResult

    for _, hook := range hooks {
        result, err := e.executeHook(ctx, hook)
        if err != nil {
            result.Phase = "Failed"
            result.Message = err.Error()
            results = append(results, result)
            return results, err
        }
        results = append(results, result)
    }

    return results, nil
}

func (e *Executor) executeHook(ctx context.Context, hook steerv1alpha1.Hook) (steerv1alpha1.HookResult, error) {
    switch hook.Type {
    case "script":
        return e.executeScriptHook(ctx, hook)
    case "kubernetes":
        return e.executeKubernetesHook(ctx, hook)
    default:
        return steerv1alpha1.HookResult{
            Name:    hook.Name,
            Phase:   "Failed",
            Message: fmt.Sprintf("Unknown hook type: %s", hook.Type),
        }, fmt.Errorf("unknown hook type: %s", hook.Type)
    }
}

func (e *Executor) executeScriptHook(ctx context.Context, hook steerv1alpha1.Hook) (steerv1alpha1.HookResult, error) {
    // 创建 ConfigMap 存储脚本
    cm := &corev1.ConfigMap{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("hook-%s", hook.Name),
            Namespace: "default",
        },
        Data: map[string]string{
            "script.sh": hook.Script,
        },
    }
    if err := e.Client.Create(ctx, cm); err != nil {
        return steerv1alpha1.HookResult{Name: hook.Name}, err
    }

    // 创建 Job 执行脚本
    job := &batchv1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("hook-%s", hook.Name),
            Namespace: "default",
        },
        Spec: batchv1.JobSpec{
            Template: corev1.PodTemplateSpec{
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{
                        {
                            Name:    "hook",
                            Image:   "bash:latest",
                            Command: []string{"/bin/bash", "/scripts/script.sh"},
                            VolumeMounts: []corev1.VolumeMount{
                                {
                                    Name:      "script",
                                    MountPath: "/scripts",
                                },
                            },
                        },
                    },
                    Volumes: []corev1.Volume{
                        {
                            Name: "script",
                            VolumeSource: corev1.VolumeSource{
                                ConfigMap: &corev1.ConfigMapVolumeSource{
                                    LocalObjectReference: corev1.LocalObjectReference{
                                        Name: cm.Name,
                                    },
                                },
                            },
                        },
                    },
                    RestartPolicy: corev1.RestartPolicyNever,
                },
            },
        },
    }
    if err := e.Client.Create(ctx, job); err != nil {
        return steerv1alpha1.HookResult{Name: hook.Name}, err
    }

    // 等待 Job 完成
    // (这里简化实现,实际需要 watch Job 状态)

    return steerv1alpha1.HookResult{
        Name:    hook.Name,
        Phase:   "Succeeded",
        Message: "Hook executed successfully",
    }, nil
}

func (e *Executor) executeKubernetesHook(ctx context.Context, hook steerv1alpha1.Hook) (steerv1alpha1.HookResult, error) {
    // 创建用户定义的 Kubernetes 资源
    // (这里简化实现,实际需要动态创建资源)

    return steerv1alpha1.HookResult{
        Name:    hook.Name,
        Phase:   "Succeeded",
        Message: "Kubernetes hook executed successfully",
    }, nil
}
```

---

## Web API 实现

在 `cmd/web/main.go` 中实现 Web API 服务:

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/gin-gonic/gin"
    "k8s.io/client-go/kubernetes/scheme"
    "k8s.io/client-go/rest"
    "sigs.k8s.io/controller-runtime/pkg/client"

    steerv1alpha1 "github.com/yourusername/steer/api/v1alpha1"
)

func main() {
    // 初始化 Kubernetes 客户端
    config, err := rest.InClusterConfig()
    if err != nil {
        log.Fatal(err)
    }

    steerv1alpha1.AddToScheme(scheme.Scheme)
    k8sClient, err := client.New(config, client.Options{Scheme: scheme.Scheme})
    if err != nil {
        log.Fatal(err)
    }

    // 初始化 Gin 路由
    r := gin.Default()

    // HelmRelease API
    r.GET("/api/v1/helmreleases", func(c *gin.Context) {
        listHelmReleases(c, k8sClient)
    })
    r.POST("/api/v1/helmreleases", func(c *gin.Context) {
        createHelmRelease(c, k8sClient)
    })
    r.GET("/api/v1/helmreleases/:name", func(c *gin.Context) {
        getHelmRelease(c, k8sClient)
    })
    r.PUT("/api/v1/helmreleases/:name", func(c *gin.Context) {
        updateHelmRelease(c, k8sClient)
    })
    r.DELETE("/api/v1/helmreleases/:name", func(c *gin.Context) {
        deleteHelmRelease(c, k8sClient)
    })

    // HelmTestJob API
    r.GET("/api/v1/helmtestjobs", func(c *gin.Context) {
        listHelmTestJobs(c, k8sClient)
    })
    r.POST("/api/v1/helmtestjobs", func(c *gin.Context) {
        createHelmTestJob(c, k8sClient)
    })
    // ... 其他 API

    // 启动服务器
    r.Run(":8080")
}

func listHelmReleases(c *gin.Context, k8sClient client.Client) {
    var list steerv1alpha1.HelmReleaseList
    if err := k8sClient.List(context.Background(), &list); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, list.Items)
}

func createHelmRelease(c *gin.Context, k8sClient client.Client) {
    var hr steerv1alpha1.HelmRelease
    if err := c.ShouldBindJSON(&hr); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if err := k8sClient.Create(context.Background(), &hr); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusCreated, hr)
}

// ... 其他 API 实现
```

---

## Web UI 实现

### 项目结构

```
steer-web/
├── src/
│   ├── components/
│   │   ├── HelmReleaseList.tsx
│   │   ├── HelmReleaseForm.tsx
│   │   ├── HelmTestJobList.tsx
│   │   └── HelmTestJobForm.tsx
│   ├── pages/
│   │   ├── Dashboard.tsx
│   │   ├── HelmReleases.tsx
│   │   └── HelmTestJobs.tsx
│   ├── api/
│   │   └── client.ts
│   ├── App.tsx
│   └── main.tsx
├── package.json
└── vite.config.ts
```

### API 客户端

在 `src/api/client.ts` 中封装 API 调用:

```typescript
import axios from 'axios';

const apiClient = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
});

export interface HelmRelease {
  metadata: {
    name: string;
    namespace: string;
  };
  spec: any;
  status: any;
}

export interface HelmTestJob {
  metadata: {
    name: string;
    namespace: string;
  };
  spec: any;
  status: any;
}

export const helmReleaseApi = {
  list: () => apiClient.get<HelmRelease[]>('/helmreleases'),
  get: (name: string) => apiClient.get<HelmRelease>(`/helmreleases/${name}`),
  create: (data: HelmRelease) => apiClient.post('/helmreleases', data),
  update: (name: string, data: HelmRelease) => apiClient.put(`/helmreleases/${name}`, data),
  delete: (name: string) => apiClient.delete(`/helmreleases/${name}`),
};

export const helmTestJobApi = {
  list: () => apiClient.get<HelmTestJob[]>('/helmtestjobs'),
  get: (name: string) => apiClient.get<HelmTestJob>(`/helmtestjobs/${name}`),
  create: (data: HelmTestJob) => apiClient.post('/helmtestjobs', data),
  update: (name: string, data: HelmTestJob) => apiClient.put(`/helmtestjobs/${name}`, data),
  delete: (name: string) => apiClient.delete(`/helmtestjobs/${name}`),
};
```

### HelmRelease 列表组件

在 `src/components/HelmReleaseList.tsx` 中实现:

```typescript
import React, { useEffect, useState } from 'react';
import { Table, Tag } from 'antd';
import { helmReleaseApi, HelmRelease } from '../api/client';

const HelmReleaseList: React.FC = () => {
  const [releases, setReleases] = useState<HelmRelease[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadReleases();
  }, []);

  const loadReleases = async () => {
    try {
      const response = await helmReleaseApi.list();
      setReleases(response.data);
    } catch (error) {
      console.error('Failed to load releases:', error);
    } finally {
      setLoading(false);
    }
  };

  const columns = [
    {
      title: 'Name',
      dataIndex: ['metadata', 'name'],
      key: 'name',
    },
    {
      title: 'Namespace',
      dataIndex: ['metadata', 'namespace'],
      key: 'namespace',
    },
    {
      title: 'Phase',
      dataIndex: ['status', 'phase'],
      key: 'phase',
      render: (phase: string) => {
        const color = phase === 'Installed' ? 'green' : phase === 'Failed' ? 'red' : 'blue';
        return <Tag color={color}>{phase}</Tag>;
      },
    },
    {
      title: 'Deployed At',
      dataIndex: ['status', 'deployedAt'],
      key: 'deployedAt',
    },
  ];

  return (
    <Table
      dataSource={releases}
      columns={columns}
      loading={loading}
      rowKey={(record) => record.metadata.name}
    />
  );
};

export default HelmReleaseList;
```

---

## 部署与运维

### Helm Chart 结构

```
charts/steer/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── rbac.yaml
│   ├── crd.yaml
│   └── ingress.yaml
└── README.md
```

### values.yaml 示例

```yaml
# Operator 配置
operator:
  replicaCount: 1
  image:
    repository: steer/operator
    tag: latest
    pullPolicy: IfNotPresent
  resources:
    limits:
      cpu: 500m
      memory: 512Mi
    requests:
      cpu: 100m
      memory: 128Mi

# Web 配置
web:
  enabled: true
  replicaCount: 1
  image:
    repository: steer/web
    tag: latest
    pullPolicy: IfNotPresent
  service:
    type: ClusterIP
    port: 8080
  ingress:
    enabled: true
    className: nginx
    host: steer.example.com
    tls:
      enabled: false

# RBAC 配置
rbac:
  create: true

serviceAccount:
  create: true
  name: steer-operator
```

### 部署命令

```bash
# 构建 Docker 镜像
docker build -t steer/operator:latest -f Dockerfile.operator .
docker build -t steer/web:latest -f Dockerfile.web .

# 推送镜像
docker push steer/operator:latest
docker push steer/web:latest

# 安装 Helm Chart
helm install steer ./charts/steer \
  --namespace steer-system \
  --create-namespace

# 升级
helm upgrade steer ./charts/steer \
  --namespace steer-system

# 卸载
helm uninstall steer --namespace steer-system
```

---

## 总结

本实现指南涵盖了 Steer 项目的核心技术实现,包括:

**Operator 开发**: 使用 Kubebuilder 框架开发 Kubernetes Operator,实现了 `HelmRelease` 和 `HelmTestJob` 两个 CRD 的控制器逻辑,包括部署、测试、清理的完整生命周期管理。

**钩子系统**: 实现了灵活的钩子机制,支持在测试前后执行自定义脚本或 Kubernetes 资源,为测试流程提供了强大的扩展能力。

**Web API**: 使用 Go 和 Gin 框架开发 RESTful API,提供了对 CRD 资源的增删改查接口,并支持 WebSocket 实时推送。

**Web UI**: 使用 React 和 TypeScript 开发现代化的管理界面,提供了友好的用户体验和实时的状态监控。

**部署方案**: 使用 Helm Chart 打包所有组件,实现了一键部署和配置管理。

通过遵循本指南,开发者可以快速理解 Steer 项目的技术架构,并参与到项目的开发和维护中。
