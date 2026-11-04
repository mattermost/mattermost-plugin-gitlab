# Cloud agent image for building the plugin and running the Playwright E2E
# suite against the stack in e2e/playwright/docker-compose.yml.
FROM ubuntu:24.04

ARG DEBIAN_FRONTEND=noninteractive
# Keep in sync with go.mod and .nvmrc.
ARG GO_VERSION=1.26.2
ARG NODE_VERSION=24.13.1
ARG DOCKER_VERSION=5:28.5.2-1~ubuntu.24.04~noble

RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential \
        ca-certificates \
        curl \
        fuse-overlayfs \
        git \
        gnupg \
        iptables \
        jq \
        make \
        python3 \
        sudo \
        xz-utils \
    && rm -rf /var/lib/apt/lists/*

RUN install -m 0755 -d /etc/apt/keyrings \
    && curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu noble stable" \
        > /etc/apt/sources.list.d/docker.list \
    && apt-get update && apt-get install -y --no-install-recommends \
        docker-ce=${DOCKER_VERSION} \
        docker-ce-cli=${DOCKER_VERSION} \
        containerd.io \
        docker-buildx-plugin \
        docker-compose-plugin \
    && rm -rf /var/lib/apt/lists/*

# Docker runs nested inside the agent's container.
RUN mkdir -p /etc/docker \
    && echo '{"storage-driver": "fuse-overlayfs"}' > /etc/docker/daemon.json \
    && update-alternatives --set iptables /usr/sbin/iptables-legacy \
    && update-alternatives --set ip6tables /usr/sbin/ip6tables-legacy

RUN arch="$(dpkg --print-architecture)" \
    && curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${arch}.tar.gz" | tar -C /usr/local -xz \
    && node_arch="$([ "$arch" = amd64 ] && echo x64 || echo "$arch")" \
    && mkdir -p /usr/local/node \
    && curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-${node_arch}.tar.xz" \
        | tar -C /usr/local/node -xJ --strip-components=1

RUN (id ubuntu >/dev/null 2>&1 || useradd -m -s /bin/bash ubuntu) \
    && usermod -aG docker,sudo ubuntu \
    && echo 'ubuntu ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/ubuntu \
    && chmod 0440 /etc/sudoers.d/ubuntu

USER ubuntu
ENV GOPATH=/home/ubuntu/go
ENV PATH=/usr/local/go/bin:/usr/local/node/bin:/home/ubuntu/go/bin:${PATH}
WORKDIR /home/ubuntu
