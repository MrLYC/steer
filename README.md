# Steer - Kubernetes Helm Test Operator

Steer 是一个基于 Kubernetes Operator 的 Helm 烟雾测试管理系统。它允许你定义 Helm Release 的发布流程,并自动执行测试任务,确保应用在 Kubernetes 集群中的稳定性。

## 核心特性

- **声明式管理**: 使用 CRD (`HelmRelease`, `HelmTestJob`) 定义发布和测试流程。
- **自动化测试**: 支持 `helm test` 和自定义脚本钩子。
- **灵活调度**: 支持一次性任务(支持延迟执行)和 Cron 周期性任务。
- **钩子系统**: 支持测试前后的自定义操作,可引用 CRD 字段作为环境变量。
- **Web 管理界面**: 提供可视化的 Dashboard,方便管理和监控。

## 快速开始（K8s 测试模式）

这是一个 **测试工具模式**：在同一个 Operator 进程中同时运行 Controller Manager + Web UI/API。
该模式用于快速验证/演示，不按生产形态做隔离（例如权限面/故障域/发布解耦）。

### 前置要求

- Kubernetes 集群（用于运行 operator）
- kubectl

### 运行（集群内）

operator 启动时带上 `--web` 指定监听地址即可开启 Web：

```bash
kubectl -n system exec -it deploy/controller-manager -- /manager --help
```

默认部署清单会以 `--web=:8082` 启动，并把 UI 静态文件挂载在容器内 `/static`。

访问方式（示例：port-forward）：

```bash
# Helm 安装且 releaseName=steer 时，web Service 默认为：steer-web
kubectl -n steer-system port-forward svc/steer-web 8080:80
```

然后访问：`http://localhost:8080/`（UI）和 `http://localhost:8080/api/v1/...`（API）。

### Helm 安装（推荐本地/演示）

仓库提供了一个 Helm chart：`charts/steer`，用于部署 operator/manager（而不是早期演示用的 `backend/`）。

```bash
helm upgrade --install steer charts/steer \
  -n steer-system --create-namespace \
  --set image.repository=<your-registry>/steer-operator \
  --set image.tag=<tag>
```

#### Metrics（不使用 kube-rbac-proxy）

该 Helm chart **不包含 kube-rbac-proxy**。metrics 由 manager 直接提供（默认只监听 127.0.0.1）。

- 默认：`127.0.0.1:8080`（集群内不可直接访问，最安全）
- 如需在集群内暴露（请自行配合 NetworkPolicy 等）：

```bash
helm upgrade --install steer charts/steer \
  -n steer-system --create-namespace \
  --set image.repository=<your-registry>/steer-operator \
  --set image.tag=<tag> \
  --set metrics.listenOnAllInterfaces=true \
  --set metrics.service.enabled=true
```

### 使用指南

1. 打开 Web 界面。
2. 在 **Helm Releases** 页面创建一个新的 Release。
3. 在 **Test Jobs** 页面创建一个新的测试任务,关联刚才创建的 Release。
   - 尝试设置 `Schedule Type` 为 `once` 并设置 `Delay` 为 `5s`。
4. 观察任务状态从 `Pending` -> `Running` -> `Succeeded` 的变化。
5. 点击 **Logs** 按钮查看测试结果和钩子执行情况。

## 项目结构

```
steer/
├── backend/              # 早期演示用的 mock 后端（已保留，但推荐使用 operator 内置 web 模式）
├── ui/                   # Web UI (React)
│   ├── client/           # 前端源码
│   │   ├── src/
│   │   │   ├── api/      # API 客户端
│   │   │   ├── components/# 公共组件
│   │   │   ├── pages/    # 页面组件
│   │   │   └── App.tsx   # 路由配置
│   └── package.json      # 前端依赖
└── operator/             # Kubebuilder operator（可选开启内置 web）
```

## CRD 定义

### HelmRelease

描述一个 Helm Release 的发布配置。

```yaml
apiVersion: steer.io/v1alpha1
kind: HelmRelease
metadata:
  name: nginx-example
  namespace: default
spec:
  chart:
    source: repository
    repository:
      name: hello-world
      url: https://helm.github.io/examples
      version: 0.1.0
  deployment:
    # 部署到的 namespace（UI 默认与 metadata.namespace 相同）
    namespace: default
  values:
    # chart values 的 YAML（也可留空字符串）
    inline: |
      replicaCount: 1
```

## UI 页面怎么配置（Embedded Web 测试模式）

### 1) 创建 Release（Releases 页面）

页面只需要填一个 **Namespace**：它会同时用于 `metadata.namespace` 和 `spec.deployment.namespace`。

用官方示例仓库的 chart（你给的 `https://helm.github.io/examples` 是有效的 Helm repo）：

- Name: `test`
- Namespace: `test-steer`
- Chart Name: `hello-world`
- Repository URL: `https://helm.github.io/examples`
- Version: `0.1.0`
- Values (YAML/JSON): 留空（或填 YAML 字符串）

对应的 CR（等价于 UI 创建的 payload）：

```yaml
apiVersion: steer.io/v1alpha1
kind: HelmRelease
metadata:
  name: test
  namespace: test-steer
spec:
  chart:
    source: repository
    repository:
      name: hello-world
      url: https://helm.github.io/examples
      version: 0.1.0
  deployment:
    namespace: test-steer
  values:
    inline: ""
```

> 你之前的报错通常是因为用了旧结构（`spec.chart.name/spec.chart.repository/spec.values:{}`）去打 operator 的 API。

### 2) 创建 Test Job（Test Jobs 页面）

- Name: `test-hello-world-01`
- Namespace: `test-steer`
- Release: 选择 `test-steer/test`
- Schedule Type: `once`
- Delay: `5s`

> NOTE：当前 operator 会在自身进程内直接执行 `helm test <release> -n <ns>`（方案 A），不再使用占位命令。
> 
> - **不配置 hooks** 时：不需要 `spec.test.image` / `STEER_JOB_IMAGE`。
> - **配置 hooks（script）** 时：hooks 仍通过 Kubernetes Job 执行，因此需要提供 `spec.test.image` 或设置环境变量 `STEER_JOB_IMAGE`（镜像需包含 /bin/sh 等）。

### HelmTestJob

描述一个测试任务。

```yaml
apiVersion: steer.io/v1alpha1
kind: HelmTestJob
metadata:
  name: test-nginx-01
  namespace: default
spec:
  helmReleaseRef:
    name: nginx-example
    namespace: default
  schedule:
    type: once
    delay: 5m  # 延迟 5 分钟执行
  test:
    timeout: 10m
  hooks:
    preTest:
      - name: notify-start
        type: script
        env:
          - name: RELEASE_NAME
            valueFrom:
              helmReleaseRef:
                fieldPath: metadata.name
        script: |
          echo "Starting test for $RELEASE_NAME"
```
