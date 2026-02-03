# OpenClaw Agent Base Image
# This is the foundation for all agent images - minimal and secure

FROM ubuntu:22.04 AS base

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install minimal required packages
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    bash \
    coreutils \
    jq \
    git \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Create non-root user
RUN groupadd -g 1000 openclaw && \
    useradd -u 1000 -g openclaw -m -s /bin/bash openclaw

# Create standard directories
RUN mkdir -p /opt/openclaw/tools \
             /opt/openclaw/runtime \
             /opt/openclaw/personality \
             /workspace \
    && chown -R openclaw:openclaw /opt/openclaw /workspace

# Set PATH to include tools directory
ENV PATH="/opt/openclaw/tools/bin:/opt/openclaw/runtime/bin:${PATH}"
ENV HOME="/home/openclaw"
ENV OPENCLAW_TOOLS_PATH="/opt/openclaw/tools"
ENV OPENCLAW_RUNTIME_PATH="/opt/openclaw/runtime"
ENV OPENCLAW_PERSONALITY_PATH="/opt/openclaw/personality"
ENV WORKSPACE="/workspace"

# Copy the OpenClaw runtime binary (to be built separately)
# COPY --chown=openclaw:openclaw openclaw-runtime /usr/local/bin/openclaw

# Switch to non-root user
USER openclaw
WORKDIR /workspace

# Default entrypoint
ENTRYPOINT ["/bin/bash"]
CMD ["-c", "echo 'OpenClaw Agent Ready' && sleep infinity"]
