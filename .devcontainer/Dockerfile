ARG GO_VERSION=1.18
##
## Base
##

FROM docker.io/debian:stable-slim as base
RUN apt update && apt upgrade -y --autoremove \
    && apt install -y \
        curl \
        file \
        git \
        jo \
        jq \
        time \
        unzip \
        xz-utils \
    && apt-get clean && rm -rf /var/lib/apt/lists/*
COPY bin/scurl /usr/local/bin/scurl

FROM base as just
ARG JUST_VERSION=1.2.0
RUN url="https://github.com/casey/just/releases/download/${JUST_VERSION}/just-${JUST_VERSION}-x86_64-unknown-linux-musl.tar.gz" ; \
    scurl "$url" | tar zvxf - -C /usr/local/bin just

FROM base as yq
ARG YQ_VERSION=v4.25.1
RUN url="https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_amd64" ; \
    scurl -o /usr/local/bin/yq "$url" && chmod +x /usr/local/bin/yq

##
## Go image
##

FROM docker.io/golang:${GO_VERSION} as go
ENV GOPROXY="https://goproxy.cn"
RUN apt update && apt upgrade -y --autoremove \
    && apt install -y \
        curl \
        file \
        jq \
        time \
        unzip \
        xz-utils \
    && apt-get clean && rm -rf /var/lib/apt/lists/*
RUN for p in \
    github.com/cweill/gotests/gotests@latest \
    github.com/go-delve/delve/cmd/dlv@latest \
    github.com/golangci/golangci-lint/cmd/golangci-lint@v1.46.2 \
    github.com/fatih/gomodifytags@latest \
    github.com/haya14busa/goplay/cmd/goplay@latest \
    github.com/josharian/impl@latest \
    github.com/ramya-rao-a/go-outline@latest \
    github.com/uudashr/gopkgs/v2/cmd/gopkgs@latest \
    golang.org/x/tools/gopls@latest \
    gotest.tools/gotestsum@v0.4.2 \
    ; do go install -mod=readonly "$p" ; done \
    && rm -rf /go/pkg/* /go/src/*
COPY bin/scurl /usr/local/bin/scurl
COPY --from=just /usr/local/bin/just /usr/local/bin/

##
## Other tools
##

FROM base as checksec
ARG CHECKSEC_VERSION=2.5.0
RUN url="https://raw.githubusercontent.com/slimm609/checksec.sh/${CHECKSEC_VERSION}/checksec" ; \
    scurl -o /usr/local/bin/checksec "$url" && chmod 755 /usr/local/bin/checksec

FROM base as shellcheck
ARG SHELLCHECK_VERSION=v0.8.0
RUN url="https://github.com/koalaman/shellcheck/releases/download/${SHELLCHECK_VERSION}/shellcheck-${SHELLCHECK_VERSION}.linux.x86_64.tar.xz" ; \
    scurl "$url" | tar xJvf - --strip-components=1 -C /usr/local/bin "shellcheck-${SHELLCHECK_VERSION}/shellcheck"

FROM shellcheck as actionlint
ARG ACTIONLINT_VERSION=v1.6.15
RUN url="https://github.com/rhysd/actionlint/releases/download/${ACTIONLINT_VERSION}/actionlint_${ACTIONLINT_VERSION#v}_linux_amd64.tar.gz" ; \
    scurl "$url" | tar xzvf - -C /usr/local/bin actionlint

##
## Tools: Everything needed for a development environment, minus non-root settings.
##

FROM base as tools
COPY --from=actionlint /usr/local/bin/actionlint /usr/local/bin/
COPY --from=checksec /usr/local/bin/checksec /usr/local/bin/che
COPY --from=just /usr/local/bin/just /usr/local/bin/
COPY --from=shellcheck /usr/local/bin/shellcheck /usr/local/bin/
COPY --from=yq /usr/local/bin/yq /usr/local/bin/

##
## Runtime
##

FROM docker.io/debian:stable as runtime
RUN apt update && apt upgrade -y --autoremove \
    && apt install -y \
        clang curl \
        cmake \
        file \
        jo \
        jq \
        libssl-dev \
        locales \
        lsb-release \
        npm \
        pkg-config \
        sudo \
        time \
        unzip \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

RUN sed -i 's/^# *\(en_US.UTF-8\)/\1/' /etc/locale.gen && locale-gen
RUN (echo "LC_ALL=zh_CN.UTF-8" \
    && echo "LANGUAGE=zh_CN.UTF-8") >/etc/default/locale

ARG MARKDOWNLINT_VERSION=0.4.0
RUN npm install "markdownlint-cli2@${MARKDOWNLINT_VERSION}" --global

ENV GOPATH=/go
COPY --from=go /go/bin $GOPATH/bin
COPY --from=go /usr/local/go /usr/local/go
RUN chmod 777 /go
ENV PATH=/usr/local/go/bin:$GOPATH/bin:$PATH

COPY --from=tools /usr/local/bin/* /usr/local/bin/

ENV DOCKER_BUILDKIT=1
RUN groupadd --gid=1000 code \
    && useradd --create-home --uid=1000 --gid=1000 code \
    && echo "code ALL=(root) NOPASSWD:ALL" >/etc/sudoers.d/code \
    && chmod 0440 /etc/sudoers.d/code \
    && scurl https://raw.githubusercontent.com/microsoft/vscode-dev-containers/main/script-library/docker-debian.sh | bash -s \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

ENV USER=code
ENV HOME=/home/code
USER code

ENTRYPOINT ["/usr/local/share/docker-init.sh"]
CMD ["sleep", "infinity"]