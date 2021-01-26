# istio install

## download istioctl

```shell
$ curl -L https://istio.io/downloadIstio | ISTIO_VERSION=1.7.3 sh -
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100   107  100   107    0     0     58      0  0:00:01  0:00:01 --:--:--    58
100  3896  100  3896    0     0   1683      0  0:00:02  0:00:02 --:--:-- 3804k
Downloading istio-1.7.3 from https://github.com/istio/istio/releases/download/1.7.3/istio-1.7.3-osx.tar.gz ...
Istio 1.7.3 Download Complete!

Istio has been successfully downloaded into the istio-1.7.3 folder on your system.

Next Steps:
See https://istio.io/docs/setup/kubernetes/install/ to add Istio to your Kubernetes cluster.

To configure the istioctl client tool for your workstation,
add the /tmp/istio-1.7.3/bin directory to your environment path variable with:
	 export PATH="$PATH:/tmp/istio-1.7.3/bin"

Begin the Istio pre-installation verification check by running:
	 istioctl verify-install

Need more information? Visit https://istio.io/docs/setup/kubernetes/install/
```

## install

```shell
$ ./istio-1.7.3/bin/istioctl install --set profile=minimal --set values.global.jwtPolicy=first-party-jwt --set addonComponents.grafana.enabled=false --set addonComponents.istiocoredns.enabled=false --set addonComponents.kiali.enabled=true --set addonComponents.prometheus.enabled=false --set addonComponents.tracing.enabled=false --set components.pilot.hub=docker.io/istio --set components.pilot.k8s.resources.requests.cpu=4000m --set components.pilot.k8s.resources.requests.memory=8Gi --set meshConfig.defaultConfig.binaryPath=/usr/local/bin/mosn --set meshConfig.defaultConfig.customConfigFile=/etc/istio/mosn/mosn_config_dubbo_xds.json --set meshConfig.defaultConfig.statusPort=15021 --set values.sidecarInjectorWebhook.rewriteAppHTTPProbe=false --set values.global.hub=symcn.tencentcloudcr.com/symcn --set values.global.proxy.logLevel=info --set values.kiali.hub=symcn.tencentcloudcr.com/symcn --set values.global.proxy.autoInject=disabled
Detected that your cluster does not support third party JWT authentication. Falling back to less secure first party JWT. See https://istio.io/docs/ops/best-practices/security/#configure-third-party-service-account-tokens for details.
! addonComponents.kiali.enabled is deprecated; use the samples/addons/ deployments instead
✔ Istio core installed
✔ Istiod installed
✔ Addons installed
✔ Installation complete

```

if already install istioctl

> istioctl install --set profile=minimal --set values.global.jwtPolicy=first-party-jwt --set addonComponents.grafana.enabled=false --set addonComponents.istiocoredns.enabled=false --set addonComponents.kiali.enabled=true --set addonComponents.prometheus.enabled=false --set addonComponents.tracing.enabled=false --set components.pilot.hub=docker.io/istio --set components.pilot.k8s.resources.requests.cpu=4000m --set components.pilot.k8s.resources.requests.memory=8Gi --set meshConfig.defaultConfig.binaryPath=/usr/local/bin/mosn --set meshConfig.defaultConfig.customConfigFile=/etc/istio/mosn/mosn_config_dubbo_xds.json --set meshConfig.defaultConfig.statusPort=15021 --set values.sidecarInjectorWebhook.rewriteAppHTTPProbe=false --set values.global.hub=symcn.tencentcloudcr.com/symcn --set values.global.proxy.logLevel=info --set values.kiali.hub=symcn.tencentcloudcr.com/symcn --set values.global.proxy.autoInject=disabled

## kiali config

```
kubectl create secret generic kiali -n istio-system --from-literal=username=admin --from-literal=passphrase=admin
```

## uninstall

### uninstall istio component

```shell
$ istioctl x uninstall --purge
All Istio resources will be pruned from the cluster
Proceed? (y/N) y
  Removed HorizontalPodAutoscaler:istio-system:istiod.
  Removed PodDisruptionBudget:istio-system:istiod.
  Removed Deployment:istio-system:istiod.
  Removed Deployment:istio-system:kiali.
  Removed Service:istio-system:istiod.
  Removed Service:istio-system:kiali.
  Removed ConfigMap:istio-system:istio.
  Removed ConfigMap:istio-system:istio-sidecar-injector.
  Removed ConfigMap:istio-system:kiali.
  Removed ServiceAccount:istio-system:istio-reader-service-account.
  Removed ServiceAccount:istio-system:istiod-service-account.
  Removed ServiceAccount:istio-system:kiali-service-account.
  Removed RoleBinding:istio-system:istiod-istio-system.
  Removed Role:istio-system:istiod-istio-system.
  Removed EnvoyFilter:istio-system:metadata-exchange-1.6.
  Removed EnvoyFilter:istio-system:metadata-exchange-1.7.
  Removed EnvoyFilter:istio-system:stats-filter-1.6.
  Removed EnvoyFilter:istio-system:stats-filter-1.7.
  Removed EnvoyFilter:istio-system:tcp-metadata-exchange-1.6.
  Removed EnvoyFilter:istio-system:tcp-metadata-exchange-1.7.
  Removed EnvoyFilter:istio-system:tcp-stats-filter-1.6.
  Removed EnvoyFilter:istio-system:tcp-stats-filter-1.7.
  Removed MutatingWebhookConfiguration::istio-sidecar-injector.
  Removed ValidatingWebhookConfiguration::istiod-istio-system.
  Removed ClusterRole::istio-reader-istio-system.
  Removed ClusterRole::istiod-istio-system.
  Removed ClusterRole::kiali.
  Removed ClusterRole::kiali-viewer.
  Removed ClusterRoleBinding::istio-reader-istio-system.
  Removed ClusterRoleBinding::istiod-pilot-istio-system.
  Removed ClusterRoleBinding::kiali.
object: MutatingWebhookConfiguration::istio-sidecar-injector is not being deleted because it no longer exists
  Removed MutatingWebhookConfiguration::istio-sidecar-injector.
object: ValidatingWebhookConfiguration::istiod-istio-system is not being deleted because it no longer exists
  Removed ValidatingWebhookConfiguration::istiod-istio-system.
  Removed CustomResourceDefinition::adapters.config.istio.io.
  Removed CustomResourceDefinition::attributemanifests.config.istio.io.
  Removed CustomResourceDefinition::authorizationpolicies.security.istio.io.
  Removed CustomResourceDefinition::destinationrules.networking.istio.io.
  Removed CustomResourceDefinition::envoyfilters.networking.istio.io.
  Removed CustomResourceDefinition::gateways.networking.istio.io.
  Removed CustomResourceDefinition::handlers.config.istio.io.
  Removed CustomResourceDefinition::httpapispecbindings.config.istio.io.
  Removed CustomResourceDefinition::httpapispecs.config.istio.io.
  Removed CustomResourceDefinition::instances.config.istio.io.
  Removed CustomResourceDefinition::istiooperators.install.istio.io.
  Removed CustomResourceDefinition::peerauthentications.security.istio.io.
  Removed CustomResourceDefinition::quotaspecbindings.config.istio.io.
  Removed CustomResourceDefinition::quotaspecs.config.istio.io.
  Removed CustomResourceDefinition::requestauthentications.security.istio.io.
  Removed CustomResourceDefinition::rules.config.istio.io.
  Removed CustomResourceDefinition::serviceentries.networking.istio.io.
  Removed CustomResourceDefinition::sidecars.networking.istio.io.
  Removed CustomResourceDefinition::templates.config.istio.io.
  Removed CustomResourceDefinition::virtualservices.networking.istio.io.
  Removed CustomResourceDefinition::workloadentries.networking.istio.io.
object: CustomResourceDefinition::adapters.config.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::adapters.config.istio.io.
object: CustomResourceDefinition::attributemanifests.config.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::attributemanifests.config.istio.io.
object: CustomResourceDefinition::authorizationpolicies.security.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::authorizationpolicies.security.istio.io.
object: CustomResourceDefinition::destinationrules.networking.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::destinationrules.networking.istio.io.
object: CustomResourceDefinition::envoyfilters.networking.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::envoyfilters.networking.istio.io.
object: CustomResourceDefinition::gateways.networking.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::gateways.networking.istio.io.
object: CustomResourceDefinition::handlers.config.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::handlers.config.istio.io.
object: CustomResourceDefinition::httpapispecbindings.config.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::httpapispecbindings.config.istio.io.
object: CustomResourceDefinition::httpapispecs.config.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::httpapispecs.config.istio.io.
object: CustomResourceDefinition::instances.config.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::instances.config.istio.io.
object: CustomResourceDefinition::istiooperators.install.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::istiooperators.install.istio.io.
object: CustomResourceDefinition::peerauthentications.security.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::peerauthentications.security.istio.io.
object: CustomResourceDefinition::quotaspecbindings.config.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::quotaspecbindings.config.istio.io.
object: CustomResourceDefinition::quotaspecs.config.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::quotaspecs.config.istio.io.
object: CustomResourceDefinition::requestauthentications.security.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::requestauthentications.security.istio.io.
object: CustomResourceDefinition::rules.config.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::rules.config.istio.io.
object: CustomResourceDefinition::serviceentries.networking.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::serviceentries.networking.istio.io.
object: CustomResourceDefinition::sidecars.networking.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::sidecars.networking.istio.io.
object: CustomResourceDefinition::templates.config.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::templates.config.istio.io.
object: CustomResourceDefinition::virtualservices.networking.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::virtualservices.networking.istio.io.
object: CustomResourceDefinition::workloadentries.networking.istio.io is not being deleted because it no longer exists
  Removed CustomResourceDefinition::workloadentries.networking.istio.io.
✔ Uninstall complete
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

