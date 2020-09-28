# istio install

## download istioctl

```shell
$ curl -L https://istio.io/downloadIstio | ISTIO_VERSION=1.6.10 sh -
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100   107  100   107    0     0     58      0  0:00:01  0:00:01 --:--:--    58
100  3896  100  3896    0     0   1683      0  0:00:02  0:00:02 --:--:-- 3804k
Downloading istio-1.6.10 from https://github.com/istio/istio/releases/download/1.6.10/istio-1.6.10-osx.tar.gz ...
Istio 1.6.10 Download Complete!

Istio has been successfully downloaded into the istio-1.6.10 folder on your system.

Next Steps:
See https://istio.io/docs/setup/kubernetes/install/ to add Istio to your Kubernetes cluster.

To configure the istioctl client tool for your workstation,
add the /tmp/istio-1.6.10/bin directory to your environment path variable with:
	 export PATH="$PATH:/tmp/istio-1.6.10/bin"

Begin the Istio pre-installation verification check by running:
	 istioctl verify-install

Need more information? Visit https://istio.io/docs/setup/kubernetes/install/
```

## install

```shell
$ ./istio-1.6.10/bin/istioctl manifest apply --set profile=minimal --set values.global.jwtPolicy=first-party-jwt --set addonComponents.grafana.enabled=false --set addonComponents.istiocoredns.enabled=false --set addonComponents.kiali.enabled=true --set addonComponents.prometheus.enabled=false --set addonComponents.tracing.enabled=false --set components.pilot.hub=docker.io/istio --set components.pilot.k8s.resources.requests.cpu=4000m --set components.pilot.k8s.resources.requests.memory=8Gi --set meshConfig.defaultConfig.binaryPath=/usr/local/bin/mosn --set meshConfig.defaultConfig.customConfigFile=/etc/istio/mosn/mosn_config_dubbo_xds.json --set meshConfig.defaultConfig.statusPort=15021 --set values.sidecarInjectorWebhook.rewriteAppHTTPProbe=false --set values.global.hub=symcn.tencentcloudcr.com/symcn --set values.global.proxy.logLevel=info --set values.kiali.hub=symcn.tencentcloudcr.com/symcn
Detected that your cluster does not support third party JWT authentication. Falling back to less secure first party JWT. See https://istio.io/docs/ops/best-practices/security/#configure-third-party-service-account-tokens for details.
! global.mtls.auto is deprecated; use meshConfig.enableAutoMtls instead
✔ Istio core installed
✔ Istiod installed
✔ Addons installed
✔ Installation complete

```

if already install istioctl

> istioctl manifest apply --set profile=minimal --set values.global.jwtPolicy=first-party-jwt --set addonComponents.grafana.enabled=false --set addonComponents.istiocoredns.enabled=false --set addonComponents.kiali.enabled=true --set addonComponents.prometheus.enabled=false --set addonComponents.tracing.enabled=false --set components.pilot.hub=docker.io/istio --set meshConfig.defaultConfig.binaryPath=/usr/local/bin/mosn --set meshConfig.defaultConfig.customConfigFile=/etc/istio/mosn/mosn_config_dubbo_xds.json --set meshConfig.defaultConfig.statusPort=15021 --set values.sidecarInjectorWebhook.rewriteAppHTTPProbe=false --set values.global.proxy.logLevel=info

## modify configmap

```shell
kubectl edit configmap -n istio-system istio-sidecar-injector
```

- modify `data.config.policy`: disabled

- delete `data.config.template.initContainers`

- add `data.config.template.containers.env`, such as: MOSN_ZK_ADDRESS

**finally result**

```yaml
apiVersion: v1
data:
  config: |-
    # modify
    policy: disabled
    alwaysInjectSelector:
      []
    neverInjectSelector:
      []
    injectedAnnotations:

    template: |
      rewriteAppHTTPProbe: {{ valueOrDefault .Values.sidecarInjectorWebhook.rewriteAppHTTPProbe false }}
      # initContainers delete
      containers:
      - name: istio-proxy
      {{- if contains "/" (annotation .ObjectMeta `sidecar.istio.io/proxyImage` .Values.global.proxy.image) }}
        image: "{{ annotation .ObjectMeta `sidecar.istio.io/proxyImage` .Values.global.proxy.image }}"
      {{- else }}
        image: "{{ .Values.global.hub }}/{{ .Values.global.proxy.image }}:{{ .Values.global.tag }}"
      {{- end }}
        ports:
        - containerPort: 15090
          protocol: TCP
          name: http-envoy-prom
        args:
        - proxy
        - sidecar
        - --domain
        - $(POD_NAMESPACE).svc.{{ .Values.global.proxy.clusterDomain }}
        - --serviceCluster
        {{ if ne "" (index .ObjectMeta.Labels "app") -}}
        - "{{ index .ObjectMeta.Labels `app` }}.$(POD_NAMESPACE)"
        {{ else -}}
        - "{{ valueOrDefault .DeploymentMeta.Name `istio-proxy` }}.{{ valueOrDefault .DeploymentMeta.Namespace `default` }}"
        {{ end -}}
        - --proxyLogLevel={{ annotation .ObjectMeta `sidecar.istio.io/logLevel` .Values.global.proxy.logLevel}}
        - --proxyComponentLogLevel={{ annotation .ObjectMeta `sidecar.istio.io/componentLogLevel` .Values.global.proxy.componentLogLevel}}
      {{- if .Values.global.sts.servicePort }}
        - --stsPort={{ .Values.global.sts.servicePort }}
      {{- end }}
      {{- if .Values.global.trustDomain }}
        - --trust-domain={{ .Values.global.trustDomain }}
      {{- end }}
      {{- if .Values.global.logAsJson }}
        - --log_as_json
      {{- end }}
      {{- if gt .ProxyConfig.Concurrency 0 }}
        - --concurrency
        - "{{ .ProxyConfig.Concurrency }}"
      {{- end -}}
      {{- if .Values.global.proxy.lifecycle }}
        lifecycle:
          {{ toYaml .Values.global.proxy.lifecycle | indent 4 }}
        {{- end }}
        env:
        # add env MOSN_ZK_ADDRESS
        - name: MOSN_ZK_ADDRESS
          value: 127.0.0.1:2181
        - name: JWT_POLICY
          value: {{ .Values.global.jwtPolicy }}
......
```

## kiali config

```
kubectl create secret generic kiali -n istio-system --from-literal=username=admin --from-literal=passphrase=admin
```

## uninstall

### uninstall istio component

```shell
$ istioctl manifest generate --set profile=demo | kubectl delete -f -
clusterrole.rbac.authorization.k8s.io "kiali" deleted
clusterrole.rbac.authorization.k8s.io "kiali-viewer" deleted
clusterrolebinding.rbac.authorization.k8s.io "kiali" deleted
configmap "kiali" deleted
secret "kiali" deleted
deployment.apps "kiali" deleted
service "kiali" deleted
serviceaccount "kiali-service-account" deleted
clusterrole.rbac.authorization.k8s.io "istiod-istio-system" deleted
clusterrole.rbac.authorization.k8s.io "istio-reader-istio-system" deleted
clusterrolebinding.rbac.authorization.k8s.io "istio-reader-istio-system" deleted
clusterrolebinding.rbac.authorization.k8s.io "istiod-pilot-istio-system" deleted
serviceaccount "istio-reader-service-account" deleted
serviceaccount "istiod-service-account" deleted
validatingwebhookconfiguration.admissionregistration.k8s.io "istiod-istio-system" deleted
customresourcedefinition.apiextensions.k8s.io "httpapispecs.config.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "httpapispecbindings.config.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "quotaspecs.config.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "quotaspecbindings.config.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "destinationrules.networking.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "envoyfilters.networking.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "gateways.networking.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "serviceentries.networking.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "sidecars.networking.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "virtualservices.networking.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "workloadentries.networking.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "attributemanifests.config.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "handlers.config.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "instances.config.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "rules.config.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "clusterrbacconfigs.rbac.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "rbacconfigs.rbac.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "serviceroles.rbac.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "servicerolebindings.rbac.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "authorizationpolicies.security.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "peerauthentications.security.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "requestauthentications.security.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "adapters.config.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "templates.config.istio.io" deleted
customresourcedefinition.apiextensions.k8s.io "istiooperators.install.istio.io" deleted
deployment.apps "istio-egressgateway" deleted
poddisruptionbudget.policy "istio-egressgateway" deleted
service "istio-egressgateway" deleted
serviceaccount "istio-egressgateway-service-account" deleted
deployment.apps "istio-ingressgateway" deleted
poddisruptionbudget.policy "istio-ingressgateway" deleted
role.rbac.authorization.k8s.io "istio-ingressgateway-sds" deleted
rolebinding.rbac.authorization.k8s.io "istio-ingressgateway-sds" deleted
service "istio-ingressgateway" deleted
serviceaccount "istio-ingressgateway-service-account" deleted
configmap "istio" deleted
deployment.apps "istiod" deleted
configmap "istio-sidecar-injector" deleted
mutatingwebhookconfiguration.admissionregistration.k8s.io "istio-sidecar-injector" deleted
poddisruptionbudget.policy "istiod" deleted
service "istiod" deleted
Error from server (NotFound): error when deleting "STDIN": configmaps "istio-grafana-configuration-dashboards-istio-mesh-dashboard" not found
Error from server (NotFound): error when deleting "STDIN": configmaps "istio-grafana-configuration-dashboards-istio-performance-dashboard" not found
Error from server (NotFound): error when deleting "STDIN": configmaps "istio-grafana-configuration-dashboards-istio-service-dashboard" not found
Error from server (NotFound): error when deleting "STDIN": configmaps "istio-grafana-configuration-dashboards-istio-workload-dashboard" not found
Error from server (NotFound): error when deleting "STDIN": configmaps "istio-grafana-configuration-dashboards-mixer-dashboard" not found
Error from server (NotFound): error when deleting "STDIN": configmaps "istio-grafana-configuration-dashboards-pilot-dashboard" not found
Error from server (NotFound): error when deleting "STDIN": configmaps "istio-grafana" not found
Error from server (NotFound): error when deleting "STDIN": deployments.apps "grafana" not found
Error from server (NotFound): error when deleting "STDIN": peerauthentications.security.istio.io "grafana-ports-mtls-disabled" not found
Error from server (NotFound): error when deleting "STDIN": services "grafana" not found
Error from server (NotFound): error when deleting "STDIN": clusterroles.rbac.authorization.k8s.io "prometheus-istio-system" not found
Error from server (NotFound): error when deleting "STDIN": clusterrolebindings.rbac.authorization.k8s.io "prometheus-istio-system" not found
Error from server (NotFound): error when deleting "STDIN": configmaps "prometheus" not found
Error from server (NotFound): error when deleting "STDIN": deployments.apps "prometheus" not found
Error from server (NotFound): error when deleting "STDIN": services "prometheus" not found
Error from server (NotFound): error when deleting "STDIN": serviceaccounts "prometheus" not found
Error from server (NotFound): error when deleting "STDIN": deployments.apps "istio-tracing" not found
Error from server (NotFound): error when deleting "STDIN": services "jaeger-query" not found
Error from server (NotFound): error when deleting "STDIN": services "jaeger-collector" not found
Error from server (NotFound): error when deleting "STDIN": services "jaeger-collector-headless" not found
Error from server (NotFound): error when deleting "STDIN": services "jaeger-agent" not found
Error from server (NotFound): error when deleting "STDIN": services "zipkin" not found
Error from server (NotFound): error when deleting "STDIN": services "tracing" not found
Error from server (NotFound): error when deleting "STDIN": the server could not find the requested resource (delete envoyfilters.networking.istio.io metadata-exchange-1.4)
Error from server (NotFound): error when deleting "STDIN": the server could not find the requested resource (delete envoyfilters.networking.istio.io stats-filter-1.4)
Error from server (NotFound): error when deleting "STDIN": the server could not find the requested resource (delete envoyfilters.networking.istio.io metadata-exchange-1.5)
Error from server (NotFound): error when deleting "STDIN": the server could not find the requested resource (delete envoyfilters.networking.istio.io tcp-metadata-exchange-1.5)
Error from server (NotFound): error when deleting "STDIN": the server could not find the requested resource (delete envoyfilters.networking.istio.io stats-filter-1.5)
Error from server (NotFound): error when deleting "STDIN": the server could not find the requested resource (delete envoyfilters.networking.istio.io tcp-stats-filter-1.5)
Error from server (NotFound): error when deleting "STDIN": the server could not find the requested resource (delete envoyfilters.networking.istio.io metadata-exchange-1.6)
Error from server (NotFound): error when deleting "STDIN": the server could not find the requested resource (delete envoyfilters.networking.istio.io tcp-metadata-exchange-1.6)
Error from server (NotFound): error when deleting "STDIN": the server could not find the requested resource (delete envoyfilters.networking.istio.io stats-filter-1.6)
Error from server (NotFound): error when deleting "STDIN": the server could not find the requested resource (delete envoyfilters.networking.istio.io tcp-stats-filter-1.6)
```

### delete config

```shell
$ kubectl get configmap -n istio-system -o wide | grep -v NAME | awk -F ' ' '{system("kubectl delete configmap -n istio-system "$1)}'
configmap "istio-ca-root-cert" deleted
configmap "istio-leader" deleted
configmap "istio-namespace-controller-election" deleted
configmap "istio-security" deleted
configmap "istio-validation-controller-election" deleted
```

### delete namespace

```shell
$ kubectl delete namespace istio-system
namespace "istio-system" deleted
```

