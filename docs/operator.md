## operator

### 安装

```shell

curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH) && chmod +x kubebuilder && mv kubebuilder /usr/local/bin/


# 测试
kubebuilder version


```

### 构建crd

```
接下来实现控制器逻辑。没什么复杂的，通过触发 reconciliation 请求获取 Foo 资源，从而得到 Foo 的朋友的名称。然后，列出所有和 Foo 的朋友同名的 Pod。如果找到一个或多个，将 Foo 的 happy 状态更新为 true，否则设置为 false。

注意，控制器也会对 Pod 事件做出反应。实际上，如果创建了一个新的 Pod，我们希望 Foo 资源能够相应更新其状态。这个方法将在每次发生 Pod 事件时被触发（创建、更新或删除）。然后，只有当 Pod 名称是集群中部署的某个 Foo 自定义资源的“朋友”时，才触发 Foo 控制器的 reconciliation 循环。


```


@see https://zhuanlan.zhihu.com/p/652773374


> 「构建简单的 Operator」

构建一个简单的 foo operator，除了演示 Operator 的功能之外，没有实际用处。运行以下命令初始化新项目

```shell

kubebuilder init --domain mega.crd --repo mega.crd/demo


# 可以开始使用 Kubebuilder 框架创建一个 Operator，我们从创建新的 API（组/版本）和新的 Kind（CRD）

# 创建组 && 控制器 (kind 的首字母必须大写)
kubebuilder create api --group demo --version v1 --kind Hello


# api/v1 包含 Hello CRD
# controllers 包含 Hello 控制器


#...
#... 逻辑
#...


# 更新 Operator manifest
make manifests


```

> 运行 Controller」

我们使用 Kind 设置本地 Kubernetes 集群，它很容易使用。

首先将 CRD 安装到集群中。

```shell

make install


# 查看 CRD
kubectl get crds

# 运行控制器
make run

# 如你所见，管理器启动了，然后 Foo 控制器也启动了，控制器现在正在运行并监听事件！


```


 > 测试

 ```yaml
apiVersion: demo.mega.crd/v1
kind: Hello
metadata:
  name: hello-01
spec:
  name: jack

---
apiVersion: demo.mega.crd/v1
kind: Hello
metadata:
  name: hello-02
spec:
  name: joe

 ```

 ```shell

# 测试
kubectl apply -f config/samples

# 查看状态
kubectl describe hello


```

> 创建 crd yaml


hello.yaml

```yaml

apiVersion: demo.mega.crd/v1
kind: Hello
metadata:
  name: hello-01
spec:
  name: jack

---
apiVersion: demo.mega.crd/v1
kind: Hello
metadata:
  name: hello-02
spec:
  name: joe
```

```shell

kubuectl -f hello.yaml

kubectl describe hello

```


接下来我们部署一个叫 jack 的 Pod 来观察系统的反应。


sleep.yaml


```yaml
apiVersion: v1
kind: Pod
metadata:
  name: jack
spec:
  containers:
    - name: ubuntu
#      image: ubuntu:latest
      image: alpine:latest
      # Just sleep forever
      command: [ "sleep" ]
      args: [ "infinity" ]
```


```shell

kubuectl -f sleep.yaml

kubectl describe hello

```

控制器应该捕获更新事件并触发 reconciliation 循环。 查看同名为jack的 pod hello-01 status变更

如果删除名为 jack 的 pod，自定义资源的 happy 状态将被设置为 false。


### 注意点

> 因为版本差异 watches中的第一个对象用 &corev1.Pod{} 替换 &source.Kind{Type: &corev1.Pod{}}

```golang

return ctrl.NewControllerManagedBy(mgr).
		For(&demov1.Hello{}).
		Watches(
			// ！！文档中使用如下方式
			//&source.Kind{Type: &corev1.Pod{}},
			// ！！ 报错尝试修改
			//source.Kind(mgr.GetCache(),  &corev1.Pod{}),
			// 最后生效
			&corev1.Pod{},
			handler.EnqueueRequestsFromMapFunc(r.mapPodsReqToFooReq),
		).
		Complete(r)

```

### controller

咱们已经体验过了kubebuilder的基本功能，不过实际生产环境中controller一般都会运行在kubernetes环境内，像上面这种运行在kubernetes之外的方式就不合适了，咱们来试试将其做成docker镜像然后在kubernetes环境运行

- 有个kubernetes可以访问的镜像仓库 （hub.docker.com）
- docker login 登录镜像仓库 根据提示输入hub.docker.com的帐号和密码
- 构建docker镜像并推送 make docker-build docker-push IMG=slimvalue/operator-demo:v0.0
- 部署controller 镜像到集群内 make deploy IMG=slimvalue/operator-demo:v0.0
- pod状态 k describe pod operator-controller-manager-6d569b9bd9-d6p8m
- 查看日志 kubectl logs -f imoc-operator-controller-manager-648b4877c6-4bpp9 -n imoc-operator-system  -c manager

> 问题

直接跑controller，会报错 account 权限不足, 连锁反应，导致程序崩溃

```log

E1124 09:27:23.002348       1 reflector.go:147] pkg/mod/k8s.io/client-go@v0.28.3/tools/cache/reflector.go:229: Failed to watch *v1.Pod: failed to list *v1.Pod: pods is forbidden: User "system:serviceaccount:default:operator-controller-manager" cannot list resource "pods" in API group "" at the cluster scope
2023-11-24T09:28:00Z    ERROR   Could not wait for Cache to sync        {"controller": "hello", "controllerGroup": "demo.mega.crd", "controllerKind": "Hello", "error": "failed to wait for hello caches to sync: timed out waiting for cache to be synced for Kind *v1.Hello"}
sigs.k8s.io/controller-runtime/pkg/internal/controller.(*Controller).Start.func2.1
        /go/pkg/mod/sigs.k8s.io/controller-runtime@v0.16.3/pkg/internal/controller/controller.go:203
...
...
...

```

> 解决办法

config/rbac/role.yaml 新增
```yaml

- apiGroups:
  - ""
  resources:
  - nodes
  - services
  - endpoints
  - pods
  verbs: [ "get", "list", "watch" ]
```

另外自动生成的 Makefile中的deploy，发布之前会重新generate rbac配置，所以新增 deploy-only，只发布

```shell
make deploy-only IMG=slimvalue/operator-demo:v0.0 
```
