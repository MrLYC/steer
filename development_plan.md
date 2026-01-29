# Steer 项目开发计划

本文档基于已设计的[项目架构](./architecture.md),制定了详细的开发里程碑和任务分解,旨在指导 Steer 项目的有序开发和交付。

## 开发里程碑

项目将分为四个主要里程碑,逐步实现从后端 Operator 到前端 Web 界面的完整功能。

| 里程碑 | 名称 | 核心目标 | 预计周期 |
| :--- | :--- | :--- | :--- |
| M1 | 核心 Operator 实现 | 完成 `HelmRelease` 和 `HelmTestJob` 两个 CRD 的核心控制器逻辑,实现自动化部署和测试。 | 3-4 周 |
| M2 | Web API 服务端开发 | 开发用于支撑 Web 界面的后端 API 服务,实现对 CRD 的增删改查和状态监控。 | 2-3 周 |
| M3 | Web UI 前端开发 | 开发用户友好的 Web 管理界面,实现对测试任务的可视化管理和操作。 | 3-4 周 |
| M4 | 打包、部署与文档完善 | 完成项目的 Helm Chart 打包,编写完善的用户和开发者文档,并建立 CI/CD 流程。 | 2 周 |

---

## M1: 核心 Operator 实现

**目标**: 完成 `HelmRelease` 和 `HelmTestJob` 两个 CRD 的核心控制器逻辑,实现自动化部署和测试。

| 任务 ID | 任务描述 | 详细内容 | 产出物 |
| :--- | :--- | :--- | :--- |
| **1.1** | **项目初始化** | 使用 Kubebuilder 初始化 Operator 项目骨架。 | Go 项目结构, `Makefile`, `Dockerfile` |
| **1.2** | **API 定义** | 在 `api/v1alpha1/` 目录下,根据架构设计定义 `HelmRelease` 和 `HelmTestJob` 的 Go 类型。 | `helmrelease_types.go`, `helmtestjob_types.go` |
| **1.3** | **CRD 生成与安装** | 运行 `make manifests` 和 `make install` 生成 CRD YAML 文件并安装到测试集群。 | CRD YAML 文件, Go 控制器骨架代码 |
| **1.4** | **HelmRelease 控制器** | 实现 `HelmRelease` 的 Reconcile 逻辑,包括: <br> - 调用 Helm SDK 执行 `install` 和 `upgrade`。<br> - 管理部署的生命周期(超时、重试)。<br> - 实现 `autoUninstallAfter` 自动卸载逻辑。<br> - 实时更新 `status` 字段。 | `helmrelease_controller.go` 的完整实现 |
| **1.5** | **HelmTestJob 控制器** | 实现 `HelmTestJob` 的 Reconcile 逻辑,包括: <br> - 实现 `once` (支持延迟执行) 和 `cron` 两种调度策略。<br> - 触发关联的 `HelmRelease` 部署并等待其就绪。<br> - 调用 Helm SDK 执行 `helm test`。 <br> - 收集测试结果并更新 `status`。 | `helmtestjob_controller.go` 的完整实现 |
| **1.6** | **钩子(Hook)实现** | 在 `HelmTestJob` 控制器中实现 `preTest` 和 `postTest` 钩子。<br> - **环境变量注入**: 实现从 CRD 字段向钩子容器注入环境变量的逻辑。<br> - **Script Hook**: 将脚本内容创建为 ConfigMap,并挂载到 Job Pod 中执行。<br> - **Kubernetes Hook**: 创建用户定义的 K8s 资源(如 Job)。 | 钩子执行逻辑代码 |
| **1.7** | **单元与集成测试** | 使用 `ginkgo` 和 `gomega` 编写测试用例。<br> - **单元测试**: 针对控制器中的独立函数。<br> - **集成测试**: 使用 `envtest` 启动一个临时的 `etcd` 和 `api-server` 来测试完整的 Reconcile 循环。 | 覆盖核心逻辑的测试用例 |

---

## M2: Web API 服务端开发

**目标**: 开发用于支撑 Web 界面的后端 API 服务,实现对 CRD 的增删改查和状态监控。

| 任务 ID | 任务描述 | 详细内容 | 产出物 |
| :--- | :--- | :--- | :--- |
| **2.1** | **Web 服务初始化** | 使用 Go 和 Gin (或 Echo) 框架搭建 Web API 项目。 | Go Web 项目结构 |
| **2.2** | **CRD 接口实现** | 实现对 `HelmRelease` 和 `HelmTestJob` 的 RESTful API。<br> - `GET /api/v1/helmreleases`<br> - `POST /api/v1/helmreleases`<br> - `GET /api/v1/helmreleases/:name`<br> - ... (其他所有 API) | API 路由和处理器代码 |
| **2.3** | **与 K8s API 交互** | 在 API 处理器中,使用 `client-go` 库与 Kubernetes API Server 交互,实现对 CRD 资源的增删改查。 | CRD 客户端代码 |
| **2.4** | **WebSocket 实现** | 实现 WebSocket API,用于向前端实时推送资源状态更新和测试日志。<br> - `/ws/helmreleases/:name`<br> - `/ws/helmtestjobs/:name` | WebSocket 处理器和推送逻辑 |
| **2.5** | **认证与授权** | 实现基础的 API 认证机制。<br> - **初期**: 使用静态 API Token。<br> - **后期**: 集成 OIDC/OAuth2,并与 K8s RBAC 对接。 | 认证中间件 |

---

## M3: Web UI 前端开发

**目标**: 开发用户友好的 Web 管理界面,实现对测试任务的可视化管理和操作。

| 任务 ID | 任务描述 | 详细内容 | 产出物 |
| :--- | :--- | :--- | :--- |
| **3.1** | **前端项目初始化** | 使用 Vite 初始化 Vue 3 + TypeScript 项目,并集成 TDesign 组件库。 | Vue 3 项目结构 |
| **3.2** | **API 集成** | 配置 API Client (如 `axios`) 和状态管理库 (如 `Pinia`) 来与后端 API 通信。 | API 请求封装和状态管理代码 |
| **3.3** | **HelmRelease 管理页面** | 开发 `HelmRelease` 的列表、创建、编辑和详情页面。<br> - **创建/编辑**: 使用表单生成 CRD 的 YAML 配置。<br> - **详情**: 展示 `status` 和相关事件。 | `HelmRelease` 相关的 Vue 组件和页面 |
| **3.4** | **HelmTestJob 管理页面** | 开发 `HelmTestJob` 的列表、创建、编辑和详情页面。<br> - **详情**: 重点展示测试结果、日志和钩子状态。 | `HelmTestJob` 相关的 Vue 组件和页面 |
| **3.5** | **实时更新与日志** | 集成 WebSocket 客户端,在详情页实时展示资源状态变化和 `helm test` 的滚动日志。 | WebSocket 集成代码和日志显示组件 |
| **3.6** | **Dashboard 页面** | 开发一个概览页面,展示关键指标,如: <br> - 正在运行的测试任务<br> - 近期测试成功率<br> - 活跃的 `HelmRelease` 数量 | Dashboard 组件 |

---

## M4: 打包、部署与文档完善

**目标**: 完成项目的 Helm Chart 打包,编写完善的用户和开发者文档,并建立 CI/CD 流程。

| 任务 ID | 任务描述 | 详细内容 | 产出物 |
| :--- | :--- | :--- | :--- |
| **4.1** | **Docker 镜像构建** | 优化 Operator 和 Web API 的 `Dockerfile`,实现多阶段构建以减小镜像体积。 | 优化的 `Dockerfile` |
| **4.2** | **Helm Chart 开发** | 创建一个统一的 Helm Chart,用于部署 Steer 的所有组件。<br> - 支持从 Git 仓库直接安装。<br> - Operator (Deployment)<br> - Web API (Deployment, Service)<br> - Web UI (可通过 Ingress 暴露)<br> - RBAC 资源 (ClusterRole, ClusterRoleBinding)<br> - CRD 定义 | Helm Chart 文件 (`Chart.yaml`, `values.yaml`, templates) |
| **4.3** | **用户文档编写** | 在 `docs` 目录下编写用户手册。<br> - **快速开始**: 如何安装和配置 Steer。<br> - **核心概念**: 介绍 `HelmRelease` 和 `HelmTestJob`。<br> - **使用指南**: 提供丰富的 CRD 示例和 Web 界面操作截图。 | 用户文档 (Markdown) |
| **4.4** | **开发者文档编写** | 编写开发者文档。<br> - **架构设计**: 引用 `architecture.md`。<br> - **本地开发**: 如何搭建开发环境和运行测试。<br> - **贡献指南**: 如何提交代码和 Issue。 | 开发者文档 (Markdown) |
| **4.5** | **CI/CD 流水线** | 使用 GitHub Actions 建立 CI/CD。<br> - **CI**: 提交代码时自动运行 `go test` 和 `lint`。<br> - **CD**: 创建 release tag 时自动构建 Docker 镜像并推送到仓库,打包 Helm Chart。 | `.github/workflows/` 下的 YAML 文件 |
