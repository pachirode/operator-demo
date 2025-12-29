# operator study

自定义资源和自定义控制器实现对应应用的自动化管理

### 安装 KubeBuilder

```bash
# go 版本 1.24.6
version="v4.9.0"
curl -L -o kubebuilder "https://github.com/kubernetes-sigs/kubebuilder/releases/download/${version}/kubebuilder_$(go env GOOS)_$(go env GOARCH)"
chmod +x kubebuilder
source <(kubebuilder completion bash)
```

### 初始化项目

```bash
mkdir operator-demo && cd operator-demo
# 初始化项目；项目域名，后续 CRD Group 的 Domain
kubebuilder init --domain crd.pachirode.com --repo github.com/pachirode/operator-demo
# go work use .
```

### 创建 API 对象

> 默认情况下只能创建一个 `Group`
> `kubebuilder edit --multigroup=true` 取消限制

```bash
kubebuilder create api --group core --version v1 --kind Application --namespaced=true
```

##### Controller

在 `internal/controller/application_controller.go` 中实现自定义控制器逻辑

- 使用 `NamespaceName` 查询
  - 获取不到可能是已经删除
- 使用 `DeletionTimestamp.IsZero` 判断是否删除
- 调谐逻辑，维护 `Deployment` 对象，并跟新状态

### 测试

```bash
# 根据自定义 CRD 生成 yaml
make manifests
# 部署到集群
make install
# 本地启动 controller
make run
# 构建镜像
IMG=pachirode/controller:latest PLATFORMS=linux/arm64,linux/amd64
make docker-build
make build-installer
# 使用配置
kubectl apply -f dist/install.yaml
kubectl get pod -n operator-demo-system 
NAME                                                READY   STATUS    RESTARTS   AGE
operator-demo-controller-manager-54886769f9-kgnqt   1/1     Running   0          17s
```
