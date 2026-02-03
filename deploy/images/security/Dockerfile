# OpenClaw Agent - Security Profile
# Security analysis, penetration testing, and compliance

FROM ghcr.io/openclaw/agent:base AS builder

USER root

# Install Python for security tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3.11 \
    python3.11-venv \
    python3.11-dev \
    python3-pip \
    build-essential \
    libffi-dev \
    libssl-dev \
    nmap \
    netcat-openbsd \
    dnsutils \
    whois \
    tcpdump \
    traceroute \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Create virtual environment
RUN python3.11 -m venv /opt/openclaw/runtime/python

# Install security Python packages
RUN /opt/openclaw/runtime/python/bin/pip install --no-cache-dir \
    bandit==1.7.* \
    safety==2.3.* \
    pip-audit==2.6.* \
    semgrep==1.50.* \
    checkov==3.* \
    prowler==3.* \
    scoutsuite==5.* \
    trufflehog==3.* \
    gitleaks-py==* \
    detect-secrets==1.4.* \
    requests==2.31.* \
    httpx==0.25.* \
    beautifulsoup4==4.12.* \
    lxml==4.9.* \
    cryptography==41.* \
    PyJWT==2.8.* \
    python-nmap==0.7.* \
    shodan==1.30.* \
    censys==2.2.*

# Install trivy
RUN curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /opt/openclaw/tools/bin v0.47.0

# Install grype (vulnerability scanner)
RUN curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /opt/openclaw/tools/bin v0.72.0

# Install syft (SBOM generator)
RUN curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /opt/openclaw/tools/bin v0.98.0

# Install cosign (container signing)
RUN curl -fsSL https://github.com/sigstore/cosign/releases/download/v2.2.1/cosign-linux-amd64 -o /opt/openclaw/tools/bin/cosign \
    && chmod +x /opt/openclaw/tools/bin/cosign

# Install kube-bench (Kubernetes CIS benchmark)
RUN curl -fsSL https://github.com/aquasecurity/kube-bench/releases/download/v0.6.19/kube-bench_0.6.19_linux_amd64.tar.gz | tar xz -C /tmp \
    && mv /tmp/kube-bench /opt/openclaw/tools/bin/

# Install kubesec (Kubernetes security scanning)
RUN curl -fsSL https://github.com/controlplaneio/kubesec/releases/download/v2.14.0/kubesec_linux_amd64.tar.gz | tar xz -C /tmp \
    && mv /tmp/kubesec /opt/openclaw/tools/bin/

# Install nuclei (vulnerability scanner)
RUN curl -fsSL https://github.com/projectdiscovery/nuclei/releases/download/v3.1.0/nuclei_3.1.0_linux_amd64.zip -o /tmp/nuclei.zip \
    && unzip /tmp/nuclei.zip -d /opt/openclaw/tools/bin \
    && rm /tmp/nuclei.zip

# Install httpx (HTTP toolkit)
RUN curl -fsSL https://github.com/projectdiscovery/httpx/releases/download/v1.3.7/httpx_1.3.7_linux_amd64.zip -o /tmp/httpx.zip \
    && unzip /tmp/httpx.zip -d /opt/openclaw/tools/bin \
    && rm /tmp/httpx.zip

# Fix permissions
RUN chown -R openclaw:openclaw /opt/openclaw

# Final image
FROM ghcr.io/openclaw/agent:base

LABEL org.opencontainers.image.title="OpenClaw Agent - Security"
LABEL org.opencontainers.image.description="OpenClaw agent for security analysis and compliance"
LABEL platform.openclaw.io/profile="security"

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    nmap \
    netcat-openbsd \
    dnsutils \
    whois \
    && rm -rf /var/lib/apt/lists/*

# Copy runtimes from builder
COPY --from=builder --chown=openclaw:openclaw /opt/openclaw/runtime /opt/openclaw/runtime
COPY --from=builder --chown=openclaw:openclaw /opt/openclaw/tools /opt/openclaw/tools

# Add Python and tools to PATH
ENV PATH="/opt/openclaw/tools/bin:/opt/openclaw/runtime/python/bin:${PATH}"
ENV PYTHONPATH="/workspace"
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

# Profile metadata
ENV OPENCLAW_PROFILE="security"
ENV OPENCLAW_PROFILE_VERSION="1.0.0"

USER openclaw
WORKDIR /workspace
