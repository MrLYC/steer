# Steer 项目架构设计

## 项目概述

Steer 是一个基于 Kubernetes Operator 的 Helm 烟雾测试管理系统。它通过自定义资源定义（CRD）来声明式地管理 Helm Release 的部署、测试和清理流程,并提供 Web 界面进行可视化管理。

## 核心价值

- **自动化测试环境**: 快速拉起近似生产环境的 K8S 环境进行单元测试和集成测试
- **资源自动清理**: 测试完成后自动清理资源,避免环境污染
- **声明式管理**: 通过 CRD 声明式地定义测试流程
- **可视化管理**: 提供 Web 界面,无需直接操作 YAML 配置

## 技术栈选型

### Operator 开发框架
- **Kubebuilder**: Go 语言的 Kubernetes Operator 开发框架
- **controller-runtime**: Kubernetes 控制器运行时库
- **Helm SDK**: 用于程序化调用 Helm 命令

### Web 前端
- **React + TypeScript**: 现代化前端框架
- **Ant Design / Material-UI**: UI 组件库
- **React Query**: 数据获取和状态管理

### Web 后端
- **Go + Gin/Echo**: 轻量级 Web 框架
- **Kubernetes Client-go**: 与 K8S API 交互

### 存储
- **Etcd**: 通过 K8S API Server 存储 CRD 资源
- **可选: PostgreSQL/MySQL**: 存储测试历史记录和日志

## CRD 设计规范

### 1. HelmRelease CRD

```yaml
apiVersion: steer.io/v1alpha1
kind: HelmRelease
metadata:
  name: example-release
  namespace: default
spec:
  # Chart 配置
  chart:
    # Chart 来源类型: repository, git, local
    source: git
    # Git 仓库配置
    git:
      url: https://github.com/example/charts.git
      ref: main
      path: charts/myapp
    # 或者 Helm Repository 配置
    repository:
      url: https://charts.example.com
      name: myapp
      version: 1.0.0
  
  # Values 配置
  values:
    # 内联 values
    inline: |
      replicaCount: 2
      image:
        repository: nginx
        tag: latest
    # 或者引用 ConfigMap/Secret
    valuesFrom:
      - configMapKeyRef:
          name: my-values
          key: values.yaml
  
  # 部署配置
  deployment:
    # 目标命名空间
    namespace: test-env
    # 是否创建命名空间
    createNamespace: true
    # 部署超时时间
    timeout: 5m
    # 失败重试次数
    retries: 3
    # 部署后等待时长
    waitAfterDeploy: 30s
    # 自动卸载时间(0 表示不自动卸载)
    autoUninstallAfter: 1h
  
  # 清理配置
  cleanup:
    # 是否清理命名空间
    deleteNamespace: true
    # 是否清理镜像(需要配置镜像仓库访问)
    deleteImages: false

status:
  # 当前状态: Pending, Installing, Installed, Failed, Uninstalling, Uninstalled
  phase: Installed
  # 部署时间
  deployedAt: "2026-01-28T10:00:00Z"
  # 预计卸载时间
  uninstallAt: "2026-01-28T11:00:00Z"
  # 错误信息
  message: ""
  # 重试次数
  retryCount: 0
  # Helm Release 信息
  helmRelease:
    name: example-release
    version: 1
    status: deployed
```

### 2. HelmTestJob CRD

```yaml
apiVersion: steer.io/v1alpha1
kind: HelmTestJob
metadata:
  name: example-test-job
  namespace: default
spec:
  # 关联的 HelmRelease
  helmReleaseRef:
    name: example-release
    namespace: default
  
  # 调度类型: once, cron
  schedule:
    type: once
    # Cron 表达式(仅 type=cron 时有效)
    cron: "0 2 * * *"
    # 时区
    timezone: Asia/Shanghai
  
  # 测试配置
  test:
    # helm test 超时时间
    timeout: 10m
    # 是否显示日志
    logs: true
    # 过滤特定的测试
    filter: ""
  
  # 钩子配置
  hooks:
    # 测试前钩子
    preTest:
      - name: validate-values
        type: script
        script: |
          #!/bin/bash
          echo "Validating values..."
          # 自定义校验逻辑
      - name: check-dependencies
        type: kubernetes
        kubernetes:
          apiVersion: batch/v1
          kind: Job
          spec:
            template:
              spec:
                containers:
                - name: checker
                  image: checker:latest
    
    # 测试后钩子
    postTest:
      - name: notify-result
        type: script
        script: |
          #!/bin/bash
          echo "Notifying test result..."
          # 发送通知
      - name: archive-logs
        type: script
        script: |
          #!/bin/bash
          echo "Archiving logs..."
  
  # 清理配置(覆盖 HelmRelease 的清理配置)
  cleanup:
    deleteNamespace: true
    deleteImages: true

status:
  # 当前状态: Pending, Running, Succeeded, Failed
  phase: Succeeded
  # 开始时间
  startTime: "2026-01-28T10:00:00Z"
  # 完成时间
  completionTime: "2026-01-28T10:15:00Z"
  # 测试结果
  testResults:
    - name: test-connection
      phase: Succeeded
      startedAt: "2026-01-28T10:10:00Z"
      completedAt: "2026-01-28T10:12:00Z"
      logs: |
        Testing connection...
        Connection successful!
  # 钩子执行结果
  hookResults:
    preTest:
      - name: validate-values
        phase: Succeeded
        message: "Validation passed"
    postTest:
      - name: notify-result
        phase: Succeeded
        message: "Notification sent"
  # 错误信息
  message: ""
  # 下次执行时间(仅 cron 类型)
  nextScheduleTime: "2026-01-29T02:00:00Z"
```

## 系统架构

### 组件架构

```
┌─────────────────────────────────────────────────────────────┐
│                         Web UI                               │
│                   (React + TypeScript)                       │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ HTTP/WebSocket
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Web API Server                          │
│                      (Go + Gin/Echo)                         │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  REST API        │  WebSocket      │  Authentication  │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ K8S Client
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Kubernetes API Server                      │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ Watch/CRUD
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Steer Operator                            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         HelmRelease Controller                        │  │
│  │  - Watch HelmRelease CRD                             │  │
│  │  - Deploy/Uninstall Helm Charts                      │  │
│  │  - Manage Auto-cleanup                               │  │
│  └──────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         HelmTestJob Controller                        │  │
│  │  - Watch HelmTestJob CRD                             │  │
│  │  - Schedule Test Jobs (Once/Cron)                    │  │
│  │  - Execute Hooks                                      │  │
│  │  - Run helm test                                      │  │
│  │  - Trigger Cleanup                                    │  │
│  └──────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         Cleanup Controller                            │  │
│  │  - Clean up Namespaces                               │  │
│  │  - Clean up Images (optional)                        │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ Helm SDK
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Helm / K8S Cluster                      │
└─────────────────────────────────────────────────────────────┘
```

### 控制器工作流程

#### HelmRelease Controller

1. **Watch 事件**: 监听 HelmRelease CRD 的创建、更新、删除事件
2. **Reconcile 循环**:
   - 检查当前状态与期望状态
   - 如果需要部署:
     - 创建/更新命名空间
     - 使用 Helm SDK 部署 Chart
     - 更新 Status 为 Installing → Installed
     - 如果配置了 waitAfterDeploy,等待指定时长
   - 如果需要卸载:
     - 检查是否到达 autoUninstallAfter 时间
     - 使用 Helm SDK 卸载 Release
     - 触发清理流程
     - 更新 Status 为 Uninstalling → Uninstalled
   - 失败重试逻辑:
     - 记录重试次数
     - 指数退避重试
     - 达到最大重试次数后标记为 Failed

#### HelmTestJob Controller

1. **Watch 事件**: 监听 HelmTestJob CRD 的创建、更新、删除事件
2. **调度逻辑**:
   - **Once 类型**: 创建后立即执行
   - **Cron 类型**: 使用 cron 库调度周期性任务
3. **执行流程**:
   - 更新 Status 为 Running
   - **部署阶段**:
     - 创建或引用 HelmRelease
     - 等待 HelmRelease 部署完成
     - 等待 waitAfterDeploy 时长
   - **前置钩子**:
     - 按顺序执行 preTest hooks
     - 如果任何钩子失败,标记整个 Job 为 Failed
   - **测试阶段**:
     - 执行 `helm test` 命令
     - 收集测试结果和日志
     - 更新 testResults 到 Status
   - **后置钩子**:
     - 按顺序执行 postTest hooks
     - 记录钩子执行结果
   - **清理阶段**:
     - 卸载 Helm Release
     - 删除命名空间
     - 清理镜像(如果配置)
   - 更新 Status 为 Succeeded/Failed

#### Cleanup Controller

1. **命名空间清理**:
   - 删除指定的命名空间
   - 等待所有资源被删除
2. **镜像清理**(可选):
   - 连接到镜像仓库
   - 删除测试使用的镜像标签
   - 清理悬空镜像

## Web 界面设计

### 功能模块

1. **Dashboard**:
   - 显示当前运行的测试任务
   - 显示测试成功率统计
   - 显示资源使用情况

2. **HelmRelease 管理**:
   - 列表页: 显示所有 HelmRelease,支持筛选、搜索
   - 创建页: 表单创建 HelmRelease
   - 详情页: 查看 HelmRelease 详情、状态、日志
   - 编辑页: 编辑 HelmRelease 配置

3. **HelmTestJob 管理**:
   - 列表页: 显示所有 HelmTestJob,支持筛选、搜索
   - 创建页: 表单创建 HelmTestJob
   - 详情页: 查看测试结果、日志、钩子执行情况
   - 编辑页: 编辑 HelmTestJob 配置

4. **测试历史**:
   - 显示历史测试记录
   - 支持按时间、状态筛选
   - 查看详细测试日志

5. **配置管理**:
   - 管理 Helm Repository
   - 管理 Git 凭证
   - 管理镜像仓库凭证

### API 设计

#### RESTful API

- `GET /api/v1/helmreleases` - 列出所有 HelmRelease
- `POST /api/v1/helmreleases` - 创建 HelmRelease
- `GET /api/v1/helmreleases/:name` - 获取 HelmRelease 详情
- `PUT /api/v1/helmreleases/:name` - 更新 HelmRelease
- `DELETE /api/v1/helmreleases/:name` - 删除 HelmRelease

- `GET /api/v1/helmtestjobs` - 列出所有 HelmTestJob
- `POST /api/v1/helmtestjobs` - 创建 HelmTestJob
- `GET /api/v1/helmtestjobs/:name` - 获取 HelmTestJob 详情
- `PUT /api/v1/helmtestjobs/:name` - 更新 HelmTestJob
- `DELETE /api/v1/helmtestjobs/:name` - 删除 HelmTestJob

- `GET /api/v1/helmtestjobs/:name/logs` - 获取测试日志
- `POST /api/v1/helmtestjobs/:name/trigger` - 手动触发测试

#### WebSocket API

- `/ws/helmreleases/:name` - 实时推送 HelmRelease 状态变化
- `/ws/helmtestjobs/:name` - 实时推送 HelmTestJob 状态和日志

## 部署方案

### Operator 部署

使用 Helm Chart 部署 Steer Operator:

```yaml
# values.yaml
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

rbac:
  create: true

serviceAccount:
  create: true
  name: steer-operator

# Web 界面配置
web:
  enabled: true
  port: 8080
  ingress:
    enabled: true
    host: steer.example.com
```

### 权限配置

Operator 需要以下 RBAC 权限:

- 管理 HelmRelease 和 HelmTestJob CRD
- 创建/删除 Namespace
- 创建/删除 Deployment、Service、ConfigMap、Secret 等资源
- 创建/删除 Job、Pod
- 读取 Events

## 安全考虑

1. **凭证管理**:
   - Git 凭证存储在 Secret 中
   - Helm Repository 凭证存储在 Secret 中
   - 镜像仓库凭证存储在 Secret 中

2. **权限隔离**:
   - 使用 ServiceAccount 限制 Operator 权限
   - 测试命名空间使用独立的 ServiceAccount
   - 支持 RBAC 策略限制用户操作

3. **Web 界面认证**:
   - 支持 OIDC/OAuth2 认证
   - 支持 K8S RBAC 集成
   - API Token 认证

## 可观测性

1. **日志**:
   - Operator 日志输出到 stdout
   - 测试日志存储在 CRD Status 中
   - 支持集成 ELK/Loki

2. **指标**:
   - 暴露 Prometheus 指标
   - 测试成功率、执行时长等指标
   - 资源使用情况指标

3. **事件**:
   - 关键操作产生 K8S Events
   - 支持 Webhook 通知

## 扩展性

1. **钩子系统**:
   - 支持 Script 类型钩子
   - 支持 Kubernetes Job 类型钩子
   - 未来可扩展 HTTP Webhook 类型

2. **插件机制**:
   - 支持自定义清理逻辑
   - 支持自定义测试报告格式

3. **多集群支持**:
   - 未来可支持管理多个 K8S 集群
   - 跨集群测试调度
