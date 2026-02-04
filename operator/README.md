# steer-operator
// TODO(user): Add simple overview of use/purpose

## Description

Steer operator 支持一个 **测试工具模式**：在同一个 manager 进程中同时运行 Controller Manager + Web UI/API。

- 通过 `--web` 开启内置 Web Server
- 通过 `--web-static-dir` 指定 UI 静态文件目录（容器内默认 `/static`）

该模式用于快速验证/演示，不建议用于生产。

## Getting Started

### Prerequisites
- go version v1.21.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/steer-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified. 
And it is required to have access to pull the image from the working environment. 
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/steer-operator:tag
```

#### Embedded Web (testing only)

默认部署清单会以 `--web=:8082` 启动 Web Server，并在 `operator/config/manager/service.yaml` 提供 `steer-web` Service。

Kustomize 的默认 `namespace` / `namePrefix` 定义在 `operator/config/default/kustomization.yaml`，最终的 Service 名称会叠加 `namePrefix`。

本地访问（port-forward）：

```sh
kubectl -n <namespace> port-forward svc/<namePrefix>steer-web 8080:80
```

访问：

- UI: http://localhost:8080/
- API: http://localhost:8080/api/v1

## Helm 安装（推荐本地/演示）

仓库提供 Helm chart：`charts/steer`，用于部署 operator/manager（controller-manager）。

```sh
helm upgrade --install steer ../../charts/steer \
  -n steer-system --create-namespace \
  --set image.repository=<your-registry>/steer-operator \
  --set image.tag=<tag>
```

默认会创建 `{{ fullname }}-web` Service。

- 当 releaseName=steer 时：Service 为 `steer-web`
- 其它 releaseName（例如 `foo`）：Service 为 `foo-steer-web`

### Metrics（不使用 kube-rbac-proxy）

`charts/steer` **不包含 kube-rbac-proxy**。metrics 由 manager 直接提供：

- 默认只监听 `127.0.0.1:8080`（集群内不可直接访问，最安全）
- 如需在集群内暴露（请自行配合 NetworkPolicy / ServiceMonitor 等）：

```sh
helm upgrade --install steer ../../charts/steer \
  -n steer-system --create-namespace \
  --set image.repository=<your-registry>/steer-operator \
  --set image.tag=<tag> \
  --set metrics.listenOnAllInterfaces=true \
  --set metrics.service.enabled=true
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin 
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/steer-operator:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/steer-operator/<tag or branch>/dist/install.yaml
```

## UI 页面配置（Embedded Web）

### Releases 页面（创建 HelmRelease）

只需要一个 Namespace：它会同时用于 `metadata.namespace` 和 `spec.deployment.namespace`。

推荐先用官方示例仓库的 chart 验证 UI + API 是否打通：

- Chart Name: `hello-world`
- Repository URL: `https://helm.github.io/examples`
- Version: `0.1.0`

### Test Jobs 页面（创建 HelmTestJob）

创建后选择刚才的 Release，并用 `once + delay` 触发一次运行。

> NOTE：当前 operator 会在自身进程内直接执行 `helm test <release> -n <ns>`（方案 A）。
>
> - **不配置 hooks** 时：不需要 `spec.test.image` / `STEER_JOB_IMAGE`。
> - **配置 hooks（script）** 时：hooks 仍通过 Kubernetes Job 执行，因此需要提供 `spec.test.image` 或设置环境变量 `STEER_JOB_IMAGE`。

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

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
