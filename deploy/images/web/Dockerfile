# OpenClaw Agent - Web Development Profile
# Frontend/Backend web development with Node.js ecosystem

FROM ghcr.io/openclaw/agent:base AS builder

USER root

# Install Node.js 20 LTS
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Create Node.js runtime directory
RUN mkdir -p /opt/openclaw/runtime/node \
    && npm config set prefix /opt/openclaw/runtime/node

# Install global npm packages
ENV PATH="/opt/openclaw/runtime/node/bin:${PATH}"
RUN npm install -g \
    typescript@5.* \
    ts-node@10.* \
    tsx@4.* \
    eslint@8.* \
    prettier@3.* \
    vite@5.* \
    next@14.* \
    create-react-app@5.* \
    create-next-app@14.* \
    @angular/cli@17.* \
    vue-cli@5.* \
    nuxt@3.* \
    svelte-kit@1.* \
    nest-cli@10.* \
    prisma@5.* \
    drizzle-kit@0.* \
    playwright@1.* \
    cypress@13.* \
    vitest@1.* \
    jest@29.* \
    pnpm@8.* \
    yarn@1.* \
    bun@1.* \
    && npm cache clean --force

# Install Deno
RUN curl -fsSL https://deno.land/install.sh | DENO_INSTALL=/opt/openclaw/runtime/deno sh

# Install Bun
RUN curl -fsSL https://bun.sh/install | BUN_INSTALL=/opt/openclaw/runtime/bun bash

# Fix permissions
RUN chown -R openclaw:openclaw /opt/openclaw

# Final image
FROM ghcr.io/openclaw/agent:base

LABEL org.opencontainers.image.title="OpenClaw Agent - Web"
LABEL org.opencontainers.image.description="OpenClaw agent for web development with Node.js ecosystem"
LABEL platform.openclaw.io/profile="web"

# Install Node.js runtime
RUN apt-get update && apt-get install -y --no-install-recommends nodejs npm \
    && rm -rf /var/lib/apt/lists/*

# Copy runtimes from builder
COPY --from=builder --chown=openclaw:openclaw /opt/openclaw/runtime /opt/openclaw/runtime

# Add all runtimes to PATH
ENV PATH="/opt/openclaw/runtime/node/bin:/opt/openclaw/runtime/deno/bin:/opt/openclaw/runtime/bun/bin:${PATH}"
ENV NODE_PATH="/opt/openclaw/runtime/node/lib/node_modules"

# Profile metadata
ENV OPENCLAW_PROFILE="web"
ENV OPENCLAW_PROFILE_VERSION="1.0.0"

USER openclaw
WORKDIR /workspace
