# BASE_DISTRIBUTION is used to switch between the old base distribution and distroless base images
ARG BASE_DISTRIBUTION=default

# Version is the base image version from the TLD Makefile
ARG BASE_VERSION=1.5-dev.3

# The following section is used as base image if BASE_DISTRIBUTION=default
FROM docker.io/istio/base:${BASE_VERSION} as default

# The following section is used as base image if BASE_DISTRIBUTION=distroless
FROM gcr.io/distroless/cc@sha256:f81e5db8287d66b012d874a6f7fea8da5b96d9cc509aa5a9b5d095a604d4bca1 as distroless

# TODO(https://github.com/istio/istio/issues/17656) clean up this hack
COPY --from=default /sbin/xtables-multi /sbin/iptables* /sbin/ip6tables* /sbin/ip /sbin/
COPY --from=default /usr/lib/x86_64-linux-gnu/xtables/ /usr/lib/x86_64-linux-gnu/xtables
COPY --from=default /usr/lib/x86_64-linux-gnu/ /usr/lib/x86_64-linux-gnu
COPY --from=default /etc/iproute2 /etc/iproute2

WORKDIR /

# This will build the final image based on either default or distroless from above
# hadolint ignore=DL3006
FROM ${BASE_DISTRIBUTION}

ARG proxy_version
ARG istio_version

# Install Mosn.
COPY mosn/mosn /usr/local/bin/mosn
COPY mosn/dubbo_config.json /etc/istio/proxy/envoy-rev0.json

# The pilot-agent will bootstrap Mosn.
ENTRYPOINT ["/usr/local/bin/mosn"]
