# Steer 项目任务清单

本文档提供了一个可追踪的任务清单,用于指导项目开发进度。每个任务完成后,请在对应的复选框中打勾。

## M1: 核心 Operator 实现 (3-4 周)

### 1.1 项目初始化
- [ ] 使用 Kubebuilder 初始化项目
- [ ] 配置 Go modules 和依赖
- [ ] 设置项目目录结构
- [ ] 配置 Makefile 和 Dockerfile
- [ ] 初始化 Git 仓库并推送到 GitHub

### 1.2 API 定义
- [ ] 定义 `HelmRelease` CRD 类型 (`helmrelease_types.go`)
  - [ ] 定义 `HelmReleaseSpec` 结构
  - [ ] 定义 `ChartSpec` 和相关子类型
  - [ ] 定义 `ValuesSpec` 和 `DeploymentSpec`
  - [ ] 定义 `CleanupSpec`
  - [ ] 定义 `HelmReleaseStatus` 结构
  - [ ] 添加 Kubebuilder 标记和打印列
- [ ] 定义 `HelmTestJob` CRD 类型 (`helmtestjob_types.go`)
  - [ ] 定义 `HelmTestJobSpec` 结构
  - [ ] 定义 `ScheduleSpec` 和 `TestSpec`
  - [ ] 定义 `HooksSpec` 和 `Hook` 类型
  - [ ] 定义 `HelmTestJobStatus` 结构
  - [ ] 添加 Kubebuilder 标记

### 1.3 CRD 生成与安装
- [ ] 运行 `make manifests` 生成 CRD YAML
- [ ] 运行 `make generate` 生成 DeepCopy 代码
- [ ] 搭建本地 K8S 测试集群 (kind/minikube)
- [ ] 运行 `make install` 安装 CRD 到集群
- [ ] 验证 CRD 安装成功 (`kubectl get crd`)

### 1.4 HelmRelease 控制器
- [ ] 实现 `HelmReleaseReconciler` 基础结构
- [ ] 实现 `reconcileInstall` 方法
  - [ ] 集成 Helm SDK 安装 Chart
  - [ ] 处理 Chart 来源 (Git/Repository/Local)
  - [ ] 处理 Values 配置 (Inline/ConfigMap/Secret)
  - [ ] 实现命名空间创建逻辑
  - [ ] 实现部署后等待逻辑
- [ ] 实现 `reconcileInstalled` 方法
  - [ ] 检查自动卸载时间
  - [ ] 实现定时重新调度
- [ ] 实现 `reconcileRetry` 方法
  - [ ] 实现失败重试逻辑
  - [ ] 实现指数退避
- [ ] 实现 `reconcileUninstall` 方法
  - [ ] 调用 Helm SDK 卸载 Release
  - [ ] 触发清理流程
- [ ] 实现状态更新逻辑
- [ ] 配置 RBAC 权限

### 1.5 HelmTestJob 控制器
- [ ] 实现 `HelmTestJobReconciler` 基础结构
- [ ] 实现 `reconcileOnce` 方法
  - [ ] 处理一次性任务执行
- [ ] 实现 `reconcileCron` 方法
  - [ ] 集成 cron 库解析表达式
  - [ ] 实现周期性任务调度
  - [ ] 计算下次执行时间
- [ ] 实现 `executeTest` 方法
  - [ ] 部署 HelmRelease 并等待就绪
  - [ ] 执行前置钩子
  - [ ] 调用 Helm SDK 执行 `helm test`
  - [ ] 收集测试结果和日志
  - [ ] 执行后置钩子
  - [ ] 触发清理流程
- [ ] 实现状态更新逻辑
- [ ] 配置 RBAC 权限

### 1.6 钩子(Hook)实现
- [ ] 创建 `pkg/hooks` 包
- [ ] 实现 `Executor` 结构
- [ ] 实现 Script Hook 执行器
  - [ ] 创建 ConfigMap 存储脚本
  - [ ] 创建 Job 执行脚本
  - [ ] 等待 Job 完成并收集结果
  - [ ] 清理临时资源
- [ ] 实现 Kubernetes Hook 执行器
  - [ ] 动态创建用户定义的 K8S 资源
  - [ ] 等待资源就绪
  - [ ] 收集执行结果
- [ ] 实现钩子结果收集和状态更新

### 1.7 Helm SDK 封装
- [ ] 创建 `pkg/helm` 包
- [ ] 实现 `Client` 结构
- [ ] 实现 `Install` 方法
- [ ] 实现 `Upgrade` 方法
- [ ] 实现 `Uninstall` 方法
- [ ] 实现 `Test` 方法
- [ ] 实现 `FetchChart` 方法
  - [ ] 支持从 Git 获取 Chart
  - [ ] 支持从 Helm Repository 获取 Chart
  - [ ] 支持从本地路径获取 Chart

### 1.8 清理逻辑实现
- [ ] 创建 `pkg/cleanup` 包
- [ ] 实现命名空间清理逻辑
  - [ ] 删除命名空间
  - [ ] 等待资源完全删除
- [ ] 实现镜像清理逻辑 (可选)
  - [ ] 连接镜像仓库
  - [ ] 删除测试使用的镜像标签
  - [ ] 清理悬空镜像

### 1.9 单元与集成测试
- [ ] 编写 HelmRelease 控制器单元测试
- [ ] 编写 HelmTestJob 控制器单元测试
- [ ] 编写 Helm SDK 封装单元测试
- [ ] 编写钩子执行器单元测试
- [ ] 使用 envtest 编写集成测试
  - [ ] 测试完整的 HelmRelease 生命周期
  - [ ] 测试完整的 HelmTestJob 生命周期
  - [ ] 测试钩子执行流程
- [ ] 确保测试覆盖率达到 70% 以上

---

## M2: Web API 服务端开发 (2-3 周)

### 2.1 Web 服务初始化
- [ ] 创建 `cmd/web` 目录
- [ ] 使用 Gin 框架初始化 Web 项目
- [ ] 配置日志和中间件
- [ ] 配置 CORS
- [ ] 配置健康检查端点

### 2.2 Kubernetes 客户端集成
- [ ] 初始化 controller-runtime client
- [ ] 配置 In-Cluster 和 Out-of-Cluster 模式
- [ ] 实现 CRD Scheme 注册

### 2.3 HelmRelease API 实现
- [ ] `GET /api/v1/helmreleases` - 列出所有 HelmRelease
  - [ ] 支持分页
  - [ ] 支持按命名空间过滤
  - [ ] 支持按状态过滤
- [ ] `POST /api/v1/helmreleases` - 创建 HelmRelease
  - [ ] 请求体验证
  - [ ] 错误处理
- [ ] `GET /api/v1/helmreleases/:name` - 获取 HelmRelease 详情
- [ ] `PUT /api/v1/helmreleases/:name` - 更新 HelmRelease
- [ ] `DELETE /api/v1/helmreleases/:name` - 删除 HelmRelease

### 2.4 HelmTestJob API 实现
- [ ] `GET /api/v1/helmtestjobs` - 列出所有 HelmTestJob
  - [ ] 支持分页
  - [ ] 支持按命名空间过滤
  - [ ] 支持按状态过滤
- [ ] `POST /api/v1/helmtestjobs` - 创建 HelmTestJob
- [ ] `GET /api/v1/helmtestjobs/:name` - 获取 HelmTestJob 详情
- [ ] `PUT /api/v1/helmtestjobs/:name` - 更新 HelmTestJob
- [ ] `DELETE /api/v1/helmtestjobs/:name` - 删除 HelmTestJob
- [ ] `GET /api/v1/helmtestjobs/:name/logs` - 获取测试日志
- [ ] `POST /api/v1/helmtestjobs/:name/trigger` - 手动触发测试

### 2.5 WebSocket 实现
- [ ] 实现 WebSocket 升级处理
- [ ] 实现 `/ws/helmreleases/:name` 端点
  - [ ] Watch HelmRelease 资源变化
  - [ ] 实时推送状态更新
- [ ] 实现 `/ws/helmtestjobs/:name` 端点
  - [ ] Watch HelmTestJob 资源变化
  - [ ] 实时推送状态和日志
- [ ] 实现连接管理和心跳机制

### 2.6 认证与授权
- [ ] 实现 API Token 认证中间件
- [ ] 实现 RBAC 权限检查
- [ ] (可选) 集成 OIDC/OAuth2
- [ ] (可选) 集成 K8S RBAC

### 2.7 API 文档
- [ ] 集成 Swagger/OpenAPI
- [ ] 编写 API 文档
- [ ] 提供 Postman Collection

### 2.8 Web API 测试
- [ ] 编写 API 单元测试
- [ ] 编写 API 集成测试
- [ ] 测试 WebSocket 功能

---

## M3: Web UI 前端开发 (3-4 周)

### 3.1 前端项目初始化
- [ ] 使用 Vite 创建 React + TypeScript 项目
- [ ] 集成 TailwindCSS
- [ ] 集成 Ant Design
- [ ] 配置路由 (React Router)
- [ ] 配置状态管理 (React Query)

### 3.2 API 客户端封装
- [ ] 创建 `src/api/client.ts`
- [ ] 封装 axios 实例
- [ ] 实现 HelmRelease API 调用
- [ ] 实现 HelmTestJob API 调用
- [ ] 实现错误处理和重试

### 3.3 公共组件开发
- [ ] 实现 Layout 组件
- [ ] 实现 Header 和 Sidebar
- [ ] 实现 Loading 组件
- [ ] 实现 ErrorBoundary
- [ ] 实现 YAML 编辑器组件

### 3.4 Dashboard 页面
- [ ] 设计 Dashboard 布局
- [ ] 实现统计卡片组件
  - [ ] 显示运行中的测试任务数
  - [ ] 显示测试成功率
  - [ ] 显示活跃的 HelmRelease 数量
- [ ] 实现最近测试列表
- [ ] 实现图表展示 (可选)

### 3.5 HelmRelease 管理页面
- [ ] 实现 HelmRelease 列表页
  - [ ] 表格展示
  - [ ] 搜索和过滤
  - [ ] 分页
  - [ ] 操作按钮 (编辑、删除)
- [ ] 实现 HelmRelease 创建页
  - [ ] 表单设计
  - [ ] 字段验证
  - [ ] YAML 预览
- [ ] 实现 HelmRelease 编辑页
- [ ] 实现 HelmRelease 详情页
  - [ ] 基本信息展示
  - [ ] 状态展示
  - [ ] 事件列表
  - [ ] YAML 查看

### 3.6 HelmTestJob 管理页面
- [ ] 实现 HelmTestJob 列表页
  - [ ] 表格展示
  - [ ] 搜索和过滤
  - [ ] 分页
  - [ ] 操作按钮 (编辑、删除、手动触发)
- [ ] 实现 HelmTestJob 创建页
  - [ ] 表单设计
  - [ ] 钩子配置表单
  - [ ] YAML 预览
- [ ] 实现 HelmTestJob 编辑页
- [ ] 实现 HelmTestJob 详情页
  - [ ] 基本信息展示
  - [ ] 测试结果展示
  - [ ] 钩子执行结果展示
  - [ ] 实时日志查看

### 3.7 实时更新与日志
- [ ] 实现 WebSocket 客户端
- [ ] 在详情页集成 WebSocket
- [ ] 实现实时状态更新
- [ ] 实现滚动日志显示组件
- [ ] 实现日志搜索和过滤

### 3.8 测试历史页面
- [ ] 实现测试历史列表
- [ ] 支持按时间范围筛选
- [ ] 支持按状态筛选
- [ ] 实现历史详情查看

### 3.9 前端测试
- [ ] 编写组件单元测试
- [ ] 编写页面集成测试
- [ ] 编写 E2E 测试 (可选)

---

## M4: 打包、部署与文档完善 (2 周)

### 4.1 Docker 镜像构建
- [ ] 编写 Operator Dockerfile
  - [ ] 多阶段构建
  - [ ] 优化镜像大小
- [ ] 编写 Web API Dockerfile
- [ ] 编写 Web UI Dockerfile
- [ ] 配置 .dockerignore
- [ ] 测试镜像构建

### 4.2 Helm Chart 开发
- [ ] 创建 `charts/steer` 目录
- [ ] 编写 `Chart.yaml`
- [ ] 编写 `values.yaml`
  - [ ] Operator 配置
  - [ ] Web API 配置
  - [ ] Web UI 配置
  - [ ] RBAC 配置
- [ ] 编写 Deployment 模板
- [ ] 编写 Service 模板
- [ ] 编写 Ingress 模板
- [ ] 编写 RBAC 模板
- [ ] 编写 CRD 模板
- [ ] 测试 Helm Chart 安装

### 4.3 CI/CD 流水线
- [ ] 创建 `.github/workflows` 目录
- [ ] 编写 CI 工作流
  - [ ] 代码检查 (golangci-lint)
  - [ ] 单元测试
  - [ ] 集成测试
  - [ ] 前端测试
- [ ] 编写 CD 工作流
  - [ ] 构建 Docker 镜像
  - [ ] 推送到镜像仓库
  - [ ] 打包 Helm Chart
  - [ ] 发布 GitHub Release
- [ ] 配置 GitHub Actions Secrets

### 4.4 用户文档编写
- [ ] 编写快速开始指南
  - [ ] 安装步骤
  - [ ] 基本使用示例
- [ ] 编写核心概念文档
  - [ ] HelmRelease 详解
  - [ ] HelmTestJob 详解
  - [ ] 钩子系统详解
- [ ] 编写使用指南
  - [ ] CRD 配置示例
  - [ ] Web 界面使用说明
  - [ ] 常见场景和最佳实践
- [ ] 编写故障排查指南
- [ ] 编写 FAQ

### 4.5 开发者文档编写
- [ ] 完善架构设计文档
- [ ] 编写本地开发指南
  - [ ] 环境搭建
  - [ ] 运行和调试
  - [ ] 测试执行
- [ ] 编写代码贡献指南
  - [ ] 代码规范
  - [ ] 提交规范
  - [ ] PR 流程
- [ ] 编写 API 参考文档

### 4.6 示例和演示
- [ ] 创建 `examples` 目录
- [ ] 编写基础示例
- [ ] 编写高级示例
- [ ] 录制演示视频 (可选)
- [ ] 准备演示环境

### 4.7 发布准备
- [ ] 编写 CHANGELOG
- [ ] 编写 LICENSE
- [ ] 更新 README
- [ ] 创建 GitHub Release
- [ ] 发布到 Helm Repository
- [ ] 撰写发布公告

---

## 后续优化 (可选)

### 性能优化
- [ ] 优化控制器性能
- [ ] 实现缓存机制
- [ ] 优化 API 响应时间
- [ ] 前端性能优化

### 功能增强
- [ ] 支持多集群管理
- [ ] 实现测试报告导出
- [ ] 集成更多通知渠道
- [ ] 实现测试模板功能
- [ ] 支持自定义清理策略

### 可观测性
- [ ] 集成 Prometheus 指标
- [ ] 集成 Grafana Dashboard
- [ ] 集成日志聚合 (ELK/Loki)
- [ ] 实现分布式追踪

### 安全加固
- [ ] 实现细粒度 RBAC
- [ ] 支持 Secret 加密
- [ ] 实现审计日志
- [ ] 安全扫描和漏洞修复

---

## 进度追踪

| 里程碑 | 开始日期 | 预计完成日期 | 实际完成日期 | 状态 |
| :--- | :--- | :--- | :--- | :--- |
| M1: 核心 Operator 实现 | - | - | - | 未开始 |
| M2: Web API 服务端开发 | - | - | - | 未开始 |
| M3: Web UI 前端开发 | - | - | - | 未开始 |
| M4: 打包、部署与文档完善 | - | - | - | 未开始 |

---

**说明**: 请在开始每个里程碑时填写开始日期,并在完成时更新实际完成日期和状态。
