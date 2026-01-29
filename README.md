# Steer

**Steer** 是一个基于 Kubernetes Operator 的 Helm 烟雾测试管理系统。它通过自定义资源定义 (CRD) 来声明式地管理 Helm Release 的部署、测试和清理流程,并提供 Web 界面进行可视化管理。

## 核心价值

Steer 旨在解决在 Kubernetes 环境中进行自动化测试时的以下痛点:

**自动化测试环境**: 快速拉起近似生产环境的 K8S 环境进行单元测试和集成测试。通过将应用打包为 Helm Chart,Steer 可以在隔离的命名空间中快速部署完整的测试环境。

**资源自动清理**: 测试完成后自动清理资源,避免环境污染。Steer 会在测试完成后自动卸载 Helm Release、删除命名空间,甚至可以清理测试使用的容器镜像,确保测试环境的干净和可重复性。

**声明式管理**: 通过 CRD 声明式地定义测试流程。用户只需要定义期望的测试配置,Steer 的控制器会自动处理部署、测试、清理的完整生命周期。

**可视化管理**: 提供 Web 界面,无需直接操作 YAML 配置。用户可以通过友好的 Web 界面创建、编辑、监控测试任务,实时查看测试日志和结果。

## 核心功能

### 自定义资源定义 (CRD)

Steer 定义了两个核心 CRD:

#### 1. HelmRelease

`HelmRelease` 用于描述一个 Helm Release 应该如何部署,包括:

- **Chart 配置**: 支持从 Git 仓库、Helm Repository 或本地路径获取 Chart
- **Values 配置**: 支持内联 values 或引用 ConfigMap/Secret
- **部署配置**: 包括目标命名空间、超时时间、失败重试次数、部署后等待时长
- **自动清理**: 支持配置自动卸载时间,到期后自动清理资源

#### 2. HelmTestJob

`HelmTestJob` 用于描述一个 Helm Release 部署完成后如何进行测试,包括:

- **调度类型**: 支持一次性任务和周期性任务 (Cron)
- **测试配置**: 执行 `helm test` 命令,收集测试结果和日志
- **钩子系统**: 支持在测试前后执行自定义脚本或 Kubernetes Job
- **自动清理**: 测试完成后自动清理 Helm Release 和相关资源

### Web 管理界面

Steer 提供了一个现代化的 Web 界面,包括:

- **Dashboard**: 展示测试任务概览和关键指标
- **HelmRelease 管理**: 创建、编辑、查看 HelmRelease 的状态和详情
- **HelmTestJob 管理**: 创建、编辑、查看测试任务,实时查看测试日志
- **测试历史**: 查看历史测试记录和详细日志

## 快速开始

### 前置条件

- Kubernetes 集群 (v1.20+)
- Helm 3.x
- kubectl 命令行工具

### 安装 Steer

使用 Helm 从 Git 仓库安装 Steer Operator 和 Web 界面:

```bash
# 克隆仓库
git clone https://github.com/yourusername/steer.git
cd steer

# 使用 Helm 安装
helm install steer ./charts/steer \
  --namespace steer-system \
  --create-namespace
```

或者直接从 Git 仓库安装(不需要克隆):

```bash
helm install steer \
  oci://ghcr.io/yourusername/steer/charts/steer \
  --namespace steer-system \
  --create-namespace
```

### 访问 Web 界面

安装完成后,可以通过 Ingress 或 Port Forward 访问 Web 界面:

```bash
# 使用 Port Forward
kubectl port-forward -n steer-system svc/steer-web 8080:8080

# 访问 http://localhost:8080
```

### 创建第一个测试任务

创建一个 `HelmRelease`:

```yaml
apiVersion: steer.io/v1alpha1
kind: HelmRelease
metadata:
  name: nginx-test
  namespace: default
spec:
  chart:
    repository:
      url: https://charts.bitnami.com/bitnami
      name: nginx
      version: 15.0.0
  values:
    inline: |
      replicaCount: 1
  deployment:
    namespace: test-nginx
    createNamespace: true
    timeout: 5m
    waitAfterDeploy: 30s
    autoUninstallAfter: 1h
  cleanup:
    deleteNamespace: true
```

创建一个 `HelmTestJob`:

```yaml
apiVersion: steer.io/v1alpha1
kind: HelmTestJob
metadata:
  name: nginx-test-job
  namespace: default
spec:
  helmReleaseRef:
    name: nginx-test
    namespace: default
  schedule:
    type: once
  test:
    timeout: 10m
    logs: true
  cleanup:
    deleteNamespace: true
```

应用配置:

```bash
kubectl apply -f helmrelease.yaml
kubectl apply -f helmtestjob.yaml
```

查看测试状态:

```bash
kubectl get helmtestjob nginx-test-job -o yaml
```

## 项目文档

- **[架构设计](./architecture.md)**: 详细的系统架构和技术设计
- **[开发计划](./development_plan.md)**: 项目开发的里程碑和任务分解
- **用户文档**: (待完善)
- **开发者文档**: (待完善)

## 开发路线图

项目将按照以下里程碑进行开发:

- **M1: 核心 Operator 实现** (3-4 周)
  - 完成 `HelmRelease` 和 `HelmTestJob` CRD 定义
  - 实现核心控制器逻辑
  - 实现钩子系统

- **M2: Web API 服务端开发** (2-3 周)
  - 开发 RESTful API
  - 实现 WebSocket 实时推送
  - 实现认证与授权

- **M3: Web UI 前端开发** (3-4 周)
  - 开发管理界面
  - 实现实时日志查看
  - 开发 Dashboard

- **M4: 打包、部署与文档完善** (2 周)
  - 创建 Helm Chart
  - 编写用户和开发者文档
  - 建立 CI/CD 流程

## 技术栈

- **后端**: Go, Kubebuilder, controller-runtime, Helm SDK
- **前端**: Vue 3, TypeScript, TDesign, Pinia, VueUse
- **部署**: Kubernetes, Helm, Docker

## 贡献

欢迎贡献代码、报告问题或提出建议!请查看 [贡献指南](./CONTRIBUTING.md) (待完善)。

## 许可证

本项目采用 MIT 许可证,详见 [LICENSE](./LICENSE) 文件。

## 联系方式

- **Issues**: https://github.com/yourusername/steer/issues
- **Discussions**: https://github.com/yourusername/steer/discussions

---

**Steer** - 让 Helm 测试更简单、更自动化。
