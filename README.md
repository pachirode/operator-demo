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

### 创建 Webhook

- `Validating Admission Webhook`
- `Mutating Admission Webhook`

```bash
# GVK 需要和 api 保持一致
kubebuilder create webhook --group core --version v1 --kind Application --defaulting --programmatic-validation
```

`internal/webhook/v1/application_webhook.go` 实现自定义逻辑

### 本地调试

没有 `webhook` 可以本地调试，如果添加了需要额外的配置

##### 定义 EndPoinys

通过自定义 `Endpoints` 来实现，手动将服务修改为本地

##### 端口转发

如果本地没有 `IP` 可以使用 `SSH` 进行端口转发
`ssh -N -R ip:9443:localhost:9443 root@ip`

##### 配置证书

使用 `kubebuilder` 推荐的 `cert-manager` 管理证书
默认生成的 `config/certmanager` 不生效，需要修改 `config/default/kustomization.yaml`

```bash
# 安装证书管理
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.16.2/cert-manager.yaml
# 修改完部署
make deploy

kubectl -n operator-demo-system get certificate
NAME                          READY   SECRET                AGE
operator-demo-metrics-certs   True    metrics-server-cert   71s
operator-demo-serving-cert    True    webhook-server-cert   71s

kubectl -n operator-demo-system get issuers.cert-manager.io
NAME                              READY   AGE
operator-demo-selfsigned-issuer   True    89s

kubectl -n operator-demo-system get secrets
NAME                  TYPE                DATA   AGE
metrics-server-cert   kubernetes.io/tls   3      3m6s
webhook-server-cert   kubernetes.io/tls   3      3m6s

# 自动注入证书
kubectl get mutatingwebhookconfigurations.admissionregistration.k8s.io operator-demo-mutating-webhook-configuration
NAME                                           WEBHOOKS   AGE
operator-demo-mutating-webhook-configuration   1          10m

# 证书下载到本地，用于本地启动 Webhook
mkdir -p /tmp/k8s-webhook-server/serving-certs
kubectl get secret -n operator-demo-system webhook-server-cert -o=jsonpath='{.data.tls\.crt}' |base64 -d > /tmp/k8s-webhook-server/serving-certs/tls.crt
kubectl get secret -n operator-demo-system webhook-server-cert -o=jsonpath='{.data.tls\.key}' |base64 -d > /tmp/k8s-webhook-server/serving-certs/tls.key
```

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

# Webhook 测试
kubectl apply -f test/config/expect.yaml
Error from server (Forbidden): error when creating "config/expect.yaml": admission webhook "vapplication-v1.kb.io" denied the request: invalid image name:

kubectl apply -f test/config/normal.yaml
application.core.crd.pachirode.com/demo created
```

### OwnerReference

一个 `Owner` 被清理他的子元素也会被清理，这部分由 `GC` 负责
子资源的变更可以触发 `Owner` 的 `Reconcile` 

```bash
# etcd 的 ownerReferences 是 Node
kubectl get pod -n kube-system etcd-kind-control-plane -o yaml | grep owner -A 5
  ownerReferences:
  - apiVersion: v1
    controller: true
    kind: Node
    name: kind-control-plane
    uid: b042fa9a-4489-414a-b249-b173de89de37
```