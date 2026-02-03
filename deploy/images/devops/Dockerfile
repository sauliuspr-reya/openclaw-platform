# OpenClaw Agent - DevOps Profile
# Infrastructure, CI/CD, and cloud operations

FROM ghcr.io/openclaw/agent:base AS builder

USER root

# Install DevOps tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    openssh-client \
    gnupg \
    lsb-release \
    software-properties-common \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Install kubectl
RUN curl -fsSL "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" -o /tmp/kubectl \
    && install -o root -g root -m 0755 /tmp/kubectl /opt/openclaw/tools/bin/kubectl \
    && rm /tmp/kubectl

# Install Helm
RUN curl -fsSL https://get.helm.sh/helm-v3.13.2-linux-amd64.tar.gz | tar xz -C /tmp \
    && mv /tmp/linux-amd64/helm /opt/openclaw/tools/bin/helm \
    && rm -rf /tmp/linux-amd64

# Install Terraform
RUN curl -fsSL https://releases.hashicorp.com/terraform/1.6.4/terraform_1.6.4_linux_amd64.zip -o /tmp/terraform.zip \
    && unzip /tmp/terraform.zip -d /opt/openclaw/tools/bin \
    && rm /tmp/terraform.zip

# Install AWS CLI
RUN curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o /tmp/awscliv2.zip \
    && unzip /tmp/awscliv2.zip -d /tmp \
    && /tmp/aws/install --install-dir /opt/openclaw/tools/aws-cli --bin-dir /opt/openclaw/tools/bin \
    && rm -rf /tmp/awscliv2.zip /tmp/aws

# Install Google Cloud SDK
RUN curl -fsSL https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-454.0.0-linux-x86_64.tar.gz | tar xz -C /opt/openclaw/tools \
    && /opt/openclaw/tools/google-cloud-sdk/install.sh --quiet --path-update=false

# Install Azure CLI
RUN curl -sL https://aka.ms/InstallAzureCLIDeb | bash

# Install Docker CLI (not daemon)
RUN curl -fsSL https://download.docker.com/linux/static/stable/x86_64/docker-24.0.7.tgz | tar xz -C /tmp \
    && mv /tmp/docker/docker /opt/openclaw/tools/bin/docker \
    && rm -rf /tmp/docker

# Install k9s
RUN curl -fsSL https://github.com/derailed/k9s/releases/download/v0.28.2/k9s_Linux_amd64.tar.gz | tar xz -C /tmp \
    && mv /tmp/k9s /opt/openclaw/tools/bin/k9s

# Install yq
RUN curl -fsSL https://github.com/mikefarah/yq/releases/download/v4.40.3/yq_linux_amd64 -o /opt/openclaw/tools/bin/yq \
    && chmod +x /opt/openclaw/tools/bin/yq

# Install kustomize
RUN curl -fsSL https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv5.3.0/kustomize_v5.3.0_linux_amd64.tar.gz | tar xz -C /opt/openclaw/tools/bin

# Fix permissions
RUN mkdir -p /opt/openclaw/tools/bin && chown -R openclaw:openclaw /opt/openclaw

# Final image
FROM ghcr.io/openclaw/agent:base

LABEL org.opencontainers.image.title="OpenClaw Agent - DevOps"
LABEL org.opencontainers.image.description="OpenClaw agent for infrastructure and CI/CD operations"
LABEL platform.openclaw.io/profile="devops"

# Copy tools from builder
COPY --from=builder --chown=openclaw:openclaw /opt/openclaw/tools /opt/openclaw/tools

# Additional PATH entries
ENV PATH="/opt/openclaw/tools/bin:/opt/openclaw/tools/google-cloud-sdk/bin:${PATH}"

# Profile metadata
ENV OPENCLAW_PROFILE="devops"
ENV OPENCLAW_PROFILE_VERSION="1.0.0"

USER openclaw
WORKDIR /workspace
