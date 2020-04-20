## env

istio info

```shell
citadel version: 1.4.6
galley version: 1.4.6
ingressgateway version: f288658b710d932bd4b0200728920fe3cbe0af61-dirty
pilot version: 1.4.6
policy version: 1.4.6
sidecar-injector version: 1.4.6
telemetry version: 1.4.6
data plane version: 1.5.0 (4 proxies), 1.4.6 (7 proxies)
```

install: [Support mosn as a sidecar](https://github.com/mosn/istio/pull/1)

kubernetes: v1.14.3

**this demo not use sidecar-injector, replace with multiple container add mosn proxy**

## step

`cd ${project_root}/examples/codes/dubbo/xds`

### build images

if you need yourself images, you can use this `command`, otherwise you can jump this step

```shell
$ make build-push
docker build -t "registry.cn-hangzhou.aliyuncs.com/sink-demo"/dubbo-consumer:"v0.0.1" -f ./consumer/Dockerfile .
Sending build context to Docker daemon  66.29MB
Step 1/3 : FROM openjdk:8-jre-stretch
 ---> 5557c40af992
Step 2/3 : COPY consumer/dubbo-examples-consumer-1.0.0.jar .
 ---> Using cache
 ---> 0feb080d5db2
Step 3/3 : ENTRYPOINT ["java", "-jar", "dubbo-examples-consumer-1.0.0.jar"]
 ---> Using cache
 ---> 1a4f4f9ee054
Successfully built 1a4f4f9ee054
Successfully tagged registry.cn-hangzhou.aliyuncs.com/sink-demo/dubbo-consumer:v0.0.1
......
```

_you can replace with your IMAGE_REPO_

### start provider

create dubbo-app namespace

```shell
kubectl create namespace dubbo-app
```

```shell
kubectl apply -f provider/install.yaml
```

_if you replace with your IMAGE_REPO, you should modify the install.yaml_

### create serviceentry

get provider ip information

```shell
$ kubectl get pods -n dubbo-app -o wide
NAME                        READY   STATUS    RESTARTS   AGE   IP             NODE            NOMINATED NODE   READINESS GATES
provider-f489bcdc6-548nf    2/2     Running   0          37m   10.13.160.40   xxxxxxxxxxxxx   <none>           <none>
provider-f489bcdc6-bhsnn    2/2     Running   0          37m   10.13.160.93   xxxxxxxxxxxxx   <none>           <none>
provider-f489bcdc6-fjpmd    2/2     Running   0          37m   10.13.160.4    xxxxxxxxxxxxx   <none>           <none>
```

_the node ip replace xxxxxxxxxxxxx_

modify serviceentry.yaml endpoints, such as:

```yaml
endpoints:
  - address: 10.13.160.40
  - address: 10.13.160.93
  - address: 10.13.160.4
```

create serviceentry

```shell
kubectl apply -f serviceentry.yaml
```

### start consumer

```shell
kubectl apply -f consumer/install.yaml
```

_if you replace with your IMAGE_REPO, you should modify the install.yaml_

### result

if it good well, you can get this result

```shell
$ kubectl logs -f -n dubbo-app consumer-7659dfcff5-mxtfn -c consumer
[20/04/20 08:00:30:549 UTC] main  INFO logger.LoggerFactory: using logger: org.apache.dubbo.common.logger.log4j.Log4jLoggerAdapter
current port:20881
[20/04/20 08:00:30:818 UTC] main  WARN config.AbstractConfig:  [DUBBO] There's no valid metadata config found, if you are using the simplified mode of registry url, please make sure you have a metadata address configured properly., dubbo version: 2.7.3, current host: 10.13.160.70
[20/04/20 08:00:31:126 UTC] main  INFO transport.AbstractClient:  [DUBBO] Succeed connect to server /10.13.160.70:20881 from NettyClient 10.13.160.70 using dubbo version 2.7.3, channel is NettyChannel [channel=[id: 0x653dcf70, L:/10.13.160.70:53222 - R:/10.13.160.70:20881]], dubbo version: 2.7.3, current host: 10.13.160.70
[20/04/20 08:00:31:126 UTC] main  INFO transport.AbstractClient:  [DUBBO] Start NettyClient consumer-7659dfcff5-mxtfn/10.13.160.70 connect to the server /10.13.160.70:20881, dubbo version: 2.7.3, current host: 10.13.160.70
[20/04/20 08:00:31:170 UTC] main  INFO config.AbstractConfig:  [DUBBO] Refer dubbo service mosn.io.dubbo.DemoService from url dubbo://127.0.0.1:20881/mosn.io.dubbo.DemoService?application=dubbo-examples-consumer&generic=false&interface=mosn.io.dubbo.DemoService&lazy=false&pid=1&qos.enable=false&register.ip=10.13.160.70&remote.application=&side=consumer&sticky=false, dubbo version: 2.7.3, current host: 10.13.160.70
Hello MOSN, response from provider: 10.13.160.4:20880
Hello MOSN, response from provider: 10.13.160.4:20880
Hello MOSN, response from provider: 10.13.160.40:20880
Hello MOSN, response from provider: 10.13.160.4:20880
Hello MOSN, response from provider: 10.13.160.40:20880
Hello MOSN, response from provider: 10.13.160.93:20880
Hello MOSN, response from provider: 10.13.160.40:20880
Hello MOSN, response from provider: 10.13.160.93:20880
Hello MOSN, response from provider: 10.13.160.4:20880
Hello MOSN, response from provider: 10.13.160.93:20880
Hello MOSN, response from provider: 10.13.160.4:20880
```

![image](https://github.com/champly/mosn/blob/feature-istio-dubbo_adapter/configs/img/result.png)
