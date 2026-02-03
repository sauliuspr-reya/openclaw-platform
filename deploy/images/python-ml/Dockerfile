# OpenClaw Agent - Python ML Profile
# Optimized for machine learning, data science, and Python development

FROM ghcr.io/openclaw/agent:base AS builder

USER root

# Install Python and development dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3.11 \
    python3.11-venv \
    python3.11-dev \
    python3-pip \
    build-essential \
    libffi-dev \
    libssl-dev \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Create virtual environment
RUN python3.11 -m venv /opt/openclaw/runtime/python

# Install ML/Data Science packages
RUN /opt/openclaw/runtime/python/bin/pip install --no-cache-dir \
    numpy==1.26.* \
    pandas==2.1.* \
    scikit-learn==1.3.* \
    scipy==1.11.* \
    matplotlib==3.8.* \
    seaborn==0.13.* \
    jupyter==1.0.* \
    ipython==8.* \
    requests==2.31.* \
    httpx==0.25.* \
    pydantic==2.5.* \
    python-dotenv==1.0.* \
    PyYAML==6.0.* \
    rich==13.* \
    typer==0.9.* \
    pytest==7.* \
    black==23.* \
    ruff==0.1.*

# Optional: Install PyTorch CPU-only (lighter weight)
RUN /opt/openclaw/runtime/python/bin/pip install --no-cache-dir \
    torch==2.1.* --index-url https://download.pytorch.org/whl/cpu \
    transformers==4.35.* \
    tokenizers==0.15.*

# Fix permissions
RUN chown -R openclaw:openclaw /opt/openclaw

# Final image
FROM ghcr.io/openclaw/agent:base

LABEL org.opencontainers.image.title="OpenClaw Agent - Python ML"
LABEL org.opencontainers.image.description="OpenClaw agent optimized for machine learning and data science"
LABEL platform.openclaw.io/profile="python-ml"

# Copy Python runtime from builder
COPY --from=builder --chown=openclaw:openclaw /opt/openclaw/runtime/python /opt/openclaw/runtime/python

# Add Python to PATH
ENV PATH="/opt/openclaw/runtime/python/bin:${PATH}"
ENV PYTHONPATH="/workspace"
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

# Profile metadata
ENV OPENCLAW_PROFILE="python-ml"
ENV OPENCLAW_PROFILE_VERSION="1.0.0"

USER openclaw
WORKDIR /workspace
