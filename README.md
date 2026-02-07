# Steer - Helm 测试护栏（Guardrails）

Steer 是一种为基于 Helm 测试的工作流提供自动化生命周期管理的工具。它通过一组 Kyverno 集群策略（ClusterPolicies）实现，为临时测试环境提供强大的“护栏”，确保资源不会被过度占用，并自动记录测试历史。

该项目不再是早期的 Operator 实现，而是完全基于 Kyverno 策略的轻量级 Helm Chart。

## 核心功能

- **命名空间自动清理**：为匹配特定标签的命名空间设置一个可配置的“存活时间”（TTL），超时后自动删除，避免资源泄漏。
- **测试完成即清理**：当 `helm test` 创建的钩子资源（如 Pod 或 Job）被删除时，自动触发对应命名空间的清理，实现即时回收。
- **测试历史记录**：在命名空间被清理前，自动捕获并记录其中关键资源（如 Pods, Deployments, Jobs 等）的状态，并将其保存为一个 ConfigMap，存放在指定的历史记录命名空间中。
- **历史记录自动归档**：为生成的历史记录 ConfigMap 也设置 TTL，实现历史数据的自动归档清理。
- **高度可配置**：所有功能均可通过 Helm `values.yaml` 文件进行开关和配置，包括命名空间选择器、TTL 时长、历史记录范围等。
- **原生集成 Kyverno**：作为一个 Helm Chart，它可以选择性地将 Kyverno 作为子 Chart 一并部署，或依赖于集群中已有的 Kyverno 实例。

## 工作原理

Steer 的核心是一系列协同工作的 Kyverno `ClusterPolicy` 资源，它们共同实现了一套完整的自动化流程：

1.  **识别目标命名空间**：用户通过为一个命名空间添加特定标签（默认为 `steer.io/managed: "true"`）来将其纳入 Steer 的管理范围。

2.  **设置初始 TTL**：一个 `Mutate` 策略会捕获新创建的受管命名空间，并自动为其添加一个 `cleanup.kyverno.io/ttl` 标签，其值由用户配置（默认为 `2h`）。这为命名空间设定了最终的“生命期限”。

3.  **监控测试钩子**：另一个 `Mutate` 策略会持续监控受管命名空间中带有 `helm.sh/hook: test` 注解的 Pod。当这类 Pod 被删除时（通常意味着 `helm test` 运行结束），该策略会立即将对应命名空间的 `cleanup.kyverno.io/ttl` 标签更新为一个较短的值（默认 `5s`，可配置），从而“提早”触发清理流程。

4.  **捕获并记录历史**：一个 `Generate` 策略会在命名空间的 `cleanup.kyverno.io/ttl` 标签被设置或更新时触发。它利用 Kyverno 的 `apiCall` 上下文变量功能，实时查询该命名空间内所有预设类型的工作负载资源，并将这些资源的状态（名称、状态等）整理成一个 JSON 字符串。

5.  **生成历史文件**：该 `Generate` 策略随后在指定的历史记录命名空间（默认为 Steer 的发布命名空间）中创建一个与被清理命名空间同名的 ConfigMap。资源状态的 JSON 字符串被存入此 ConfigMap 的 `data` 字段。同时，这个新创建的 ConfigMap 自身也被打上了一个 `cleanup.kyverno.io/ttl` 标签（默认为 `24h`），以便在未来自动删除。

6.  **自动清理**：最后，Kyverno 内置的 `cleanup-controller` 会根据 `cleanup.kyverno.io/ttl` 标签，在指定时间到达后，安全地删除命名空间和历史记录 ConfigMap。

这个设计巧妙地利用了 Kyverno 的声明式策略和自动化能力，形成了一个无需外部控制器或 Operator 的、完全在集群内部自洽运行的护栏系统。

## 快速开始

### 前提条件

- 一个正在运行的 Kubernetes 集群。
- `kubectl` 和 `helm` 已安装并配置。
- 集群中已安装 Kyverno（v1.9+）。如果未安装，可以在部署 Steer 时通过 `values.yaml` 启用 Kyverno 子 Chart 的安装。

### 安装

1.  **添加 Helm 仓库（如果需要发布）**

    ```bash
    # 此步骤仅在未来图表发布到公共仓库后需要
    # helm repo add steer <repo-url>
    # helm repo update
    ```

2.  **安装 Steer Chart**

    将此仓库克隆到本地，然后使用 Helm 安装。

    ```bash
    git clone https://github.com/<your-username>/steer.git
    cd steer

    # 安装到 steer 命名空间（建议加上 --dependency-update 以自动拉取子 chart 依赖）
    helm install steer ./charts/steer -n steer --create-namespace --dependency-update
    ```

3.  **安装并启用 Kyverno（如果集群中没有）**

    如果你的集群尚未安装 Kyverno，可以使用 `examples/values-with-kyverno.yaml` 文件来同时部署 Steer 和 Kyverno。

    ```bash
    helm install steer ./charts/steer -n steer --create-namespace --dependency-update \
      -f examples/values-with-kyverno.yaml
    ```

### 使用示例

1.  **创建一个用于测试的命名空间，并打上 Steer 管理标签。**

    ```bash
    kubectl create namespace my-helm-test
    kubectl label namespace my-helm-test steer.io/managed=true
    ```

2.  **在该命名空间中运行一个带有 `helm test` 钩子的 Chart。**

    Helm 会在这个命名空间中创建一个或多个带有 `helm.sh/hook: test` 注解的 Pod。

3.  **观察效果**

    - **命名空间 TTL**：检查 `my-helm-test` 命名空间，你会发现它被自动添加了 `cleanup.kyverno.io/ttl: 2h` 标签。
      ```bash
      kubectl get namespace my-helm-test --show-labels
      ```

    - **测试完成**：当 `helm test` 结束并删除测试 Pod 后，再次检查该命名空间，`cleanup.kyverno.io/ttl` 标签会被更新为一个较短的值（默认 `5s`，可配置）。

    - **历史记录**：在命名空间被删除后，检查 Steer 所在的命名空间（或 `values.yaml` 中指定的历史命名空间），会发现一个名为 `my-helm-test` 的 ConfigMap。
      ```bash
      # 假设 steer 安装在 "steer" 命名空间
      kubectl get configmap my-helm-test -n steer -o yaml
      ```
      这个 ConfigMap 的 `data` 字段中包含了 `my-helm-test` 命名空间被删除前的资源快照。

## 配置

所有配置项都在 `charts/steer/values.yaml` 文件中，并有详细注释。主要配置包括：

| 参数 | 描述 | 默认值 |
|---|---|---|
| `namespaceSelector.key` | 用于识别受管命名空间的标签键。 | `steer.io/managed` |
| `namespaceSelector.value` | 用于识别受管命名空间的标签值。 | `true` |
| `namespaceTTL` | 受管命名空间的默认存活时间。 | `2h` |
| `cleanupOnTestComplete.enabled` | 是否在 Helm 测试钩子资源删除后立即清理命名空间。 | `true` |
| `cleanupOnTestComplete.ttl` | 测试钩子资源删除后用于加速清理的 TTL（建议较短）。 | `5s` |
| `history.enabled` | 是否启用测试历史记录。 | `true` |
| `history.namespace` | 存储历史记录 ConfigMap 的命名空间。 | `{{ .Release.Namespace }}` |
| `history.ttl` | 历史记录 ConfigMap 的存活时间。 | `24h` |
| `history.resources` | 一个对象数组，定义了需要记录哪些资源类型及其 API 路径。 | 预设的 Kubernetes 工作负载 |
| `kyverno.enabled` | 是否将 Kyverno 作为子 Chart 一同部署。 | `false` |

## 贡献

欢迎通过 Pull Request 或 Issue 对项目做出贡献。在本地开发和测试时，你可以使用 `helm template` 或 `helm install --dry-run` 命令来渲染和验证模板的输出。

```bash
# 渲染模板并查看输出
helm template steer ./charts/steer -n steer --dependency-update
```
