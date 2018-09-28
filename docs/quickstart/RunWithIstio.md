## 一、准备工作

- ### [安装 kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl)

```bash
$ brew install kubernetes-cli
...
$ kubectl version
Client Version: version.Info{Major:"1", Minor:"11", GitVersion:"v1.11.2", GitCommit:"bb9ffb1654d4a729bb4cec18ff088eacc153c239", GitTreeState:"clean", BuildDate:"2018-08-08T16:31:10Z", GoVersion:"go1.10.3", Compiler:"gc", Platform:"darwin/amd64"}
Server Version: version.Info{Major:"1", Minor:"10+", GitVersion:"v1.10.1-25+cbc1f79f9924ea", GitCommit:"cbc1f79f9924ea45ae0618a3544986226abb8469", GitTreeState:"clean", BuildDate:"2018-05-07T11:43:18Z", GoVersion:"go1.9.3", Compiler:"gc", Platform:"linux/amd64"}
```

- ### 创建 k8s 集群
  - [Minikube](https://lark.alipay.com/basement-group/the-road-to-containerized/re9a6g) 
  - 购买 Aliyun (推荐新加坡节点，国内的下载镜像巨慢), GKE 等商业版，GKE 首次注册送 $300

- ### [配置 kubectl 访问集群](https://kubernetes.io/cn/docs/tasks/access-application-cluster/configure-access-multiple-clusters/)

设置完成后，通过 `kubectl cluster-info` 确保连接到正确的集群
```bash
$ kubectl config use-context default
Switched to context "default".

$ kubectl cluster-info
Kubernetes master is running at https://apiserver.56.sqaztt.alipay.net:6443
KubeDNS is running at https://apiserver.56.sqaztt.alipay.net:6443/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy

To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.
```

## 二、源码方式部署 sofa-mesh

- ### 下载 sofa-mesh 源码

```bash
$ git clone git@github.com:alipay/sofa-mesh.git
```

- ### 构建 istio 的 YAML 配置

在根目录下创建一个 `.istiorc.mk` 文件，用它来覆盖镜像的 tag 为 1.0.0

```bash
$ echo TAG="1.0.0" > .istiorc.mk
```

根目录下执行命令，将生成 istio.yaml 和 istio-auth.yaml 两个配置文件

```bash
$ sh install/updateVersion.sh
```

- install/kubernetes/istio-auth.yaml
- install/kubernetes/istio.yaml

- ### 替换 istio 的默认命名空间（测试环境公用的，避免冲突）

通过下面命令我们将默认的 ns 改为 istio-system1，新的配置放在 install/kubernetes/istio-system1.yaml

```sh
cat install/kubernetes/istio.yaml | sed s/istio-system/istio-system1/g > install/kubernetes/istio-system1.yaml
```

- ### 部署 sofa-mesh

安装 istio 的自定义资源
```bash
$ kubectl apply -f install/kubernetes/helm/istio/templates/crds.yaml
```

安装 sofa-mesh
```bash
$ kubectl apply -f install/kubernetes/istio-system1.yaml
```

- ### 验证安装

```bash
$ kubectl get svc -n istio-system1
NAME                       TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)                                 AGE
istio-citadel              ClusterIP   172.16.113.0     <none>        8060/TCP,9093/TCP                       2m
istio-egressgateway        ClusterIP   172.16.93.234    <none>        80/TCP,443/TCP                          2m
istio-galley               ClusterIP   172.16.199.113   <none>        443/TCP,9093/TCP                        2m
istio-pilot                ClusterIP   172.16.94.105    <none>        15010/TCP,15011/TCP,8080/TCP,9093/TCP   2m
istio-policy               ClusterIP   172.16.152.158   <none>        9091/TCP,15004/TCP,9093/TCP             2m
istio-sidecar-injector     ClusterIP   172.16.226.86    <none>        443/TCP                                 2m
istio-statsd-prom-bridge   ClusterIP   172.16.18.241    <none>        9102/TCP,9125/UDP                       2m
istio-telemetry            ClusterIP   172.16.200.109   <none>        9091/TCP,15004/TCP,9093/TCP,42422/TCP   2m
prometheus                 ClusterIP   172.16.157.229   <none>        9090/TCP                                2m
```

istio-system1 命名空间下的 pod 状态都是 Running 时，说明已经部署成功

```bash
$ kubectl get pods -n istio-system1
NAME                                        READY     STATUS              RESTARTS   AGE
istio-citadel-965587bb-vzmtd                0/1       ContainerCreating   0          2m
istio-cleanup-secrets-w6vnw                 0/1       ContainerCreating   0          2m
istio-egressgateway-5745bd96f-ph2jn         0/1       ContainerCreating   0          2m
istio-galley-5cb74d5b55-pjh4d               0/1       ContainerCreating   0          2m
istio-ingressgateway-65cd86fcd-m9kth        0/1       ContainerCreating   0          2m
istio-pilot-649899d9f6-gfrhb                0/1       ContainerCreating   0          2m
istio-policy-6b5d7945f6-t4hxs               0/1       ContainerCreating   0          2m
istio-sidecar-injector-58bb8694c5-hxbgz     0/1       ContainerCreating   0          2m
istio-statsd-prom-bridge-7f44bb5ddb-n8vz9   0/1       ContainerCreating   0          2m
istio-telemetry-6d94577dfc-ffs25            0/1       ContainerCreating   0          2m
prometheus-84bd4b9796-l4dxm                 0/1       ContainerCreating   0          2m
```

## 三、实验

可以试试 istio 官方的 [Bookinfo](https://istio.io/docs/examples/bookinfo/) 实验

![undefined](https://cdn.nlark.com/lark/0/2018/png/2054/1535048760569-e615ca21-1742-418d-8210-3db28bd12bf0.png) 