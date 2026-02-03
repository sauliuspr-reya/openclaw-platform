# OpenClaw Platform

A Kubernetes operator for running autonomous AI agents with hierarchical shared memory, swarm coordination, and container mode duality (immutable/dynamic).

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Kubernetes Cluster                                 │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                    NATS JetStream (Memory Layer)                       │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │ │
│  │  │   Company   │  │   Groups    │  │   Swarms    │  │ Individual  │  │ │
│  │  │   Memory    │  │   Memory    │  │   Memory    │  │   Memory    │  │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘  │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                     │                                       │
│                                     ▼                                       │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                       OpenClaw Operator                                │ │
│  │  - Reconciles OpenClawAgent/OpenClawSwarm CRDs                        │ │
│  │  - Creates Pods with personality injection                             │ │
│  │  - Manages container mode (immutable/dynamic)                          │ │
│  └───────────────────────────────────────────────────────────────────────┘ │
│                                     │                                       │
│           ┌─────────────────────────┼─────────────────────────┐            │
│           ▼                         ▼                         ▼            │
│  ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐      │
│  │  Agent Pod      │     │  Agent Pod      │     │  Agent Pod      │      │
│  │  (python-ml)    │     │  (devops)       │     │  (dynamic)      │      │
│  │  ┌───────────┐  │     │  ┌───────────┐  │     │  ┌───────────┐  │      │
│  │  │ IDENTITY  │  │     │  │ IDENTITY  │  │     │  │ IDENTITY  │  │      │
│  │  │ SOUL      │  │     │  │ SOUL      │  │     │  │ SOUL      │  │      │
│  │  └───────────┘  │     │  └───────────┘  │     │  └───────────┘  │      │
│  └─────────────────┘     └─────────────────┘     └─────────────────┘      │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Key Features

### 1. Container Mode Duality

**Immutable Mode (default)**: Pre-baked images with fixed toolsets for fast startup
- `minimal` - Basic CLI tools only
- `python-ml` - ML/Data Science (PyTorch, pandas, scikit-learn)
- `devops` - Infrastructure tools (kubectl, terraform, helm)
- `web` - Web development (Node.js, TypeScript, various frameworks)
- `data` - Data engineering (DuckDB, Spark, dbt)
- `security` - Security tools (trivy, nuclei, semgrep)

**Dynamic Mode**: Mutable containers that can install tools at runtime
- Uses writable volume at `/opt/openclaw/tools`
- Optionally persists tools via PVC
- Falls back when tasks need unusual tools

### 2. Hierarchical Memory System

Four-tier memory hierarchy backed by NATS JetStream:

| Tier | Scope | Persistence | Use Case |
|------|-------|-------------|----------|
| **Company** | All agents | Permanent | Shared knowledge, policies, tool catalog |
| **Group** | Functional teams | Long-lived | Team-specific patterns, preferences |
| **Swarm** | Task cluster | Ephemeral | Coordination, intermediate results |
| **Individual** | Single agent | Session | Working context, identity, local cache |

### 3. Personality System (IDENTITY.md / SOUL.md)

Agents have injected personalities that define:
- **Identity**: Role, expertise, responsibilities
- **Soul**: Behavioral traits, values, communication style

Loaded from ConfigMaps, Secrets, or inline in the CRD.

### 4. Swarm Coordination

Agents can work individually or in coordinated swarms:
- **Orchestrator**: Splits tasks into shards, assigns to workers, merges results
- **Workers**: Process individual shards, report progress, share discoveries

## Quick Start

### Prerequisites

- Kubernetes cluster (1.28+)
- kubectl configured
- Docker (for building images)

### Deploy

```bash
# Install CRDs
make install-crds

# Deploy NATS JetStream
make deploy-nats

# Initialize memory buckets
make init-buckets

# Deploy operator
make deploy

# Deploy example agents
make deploy-examples
```

### Create an Agent

```yaml
apiVersion: platform.openclaw.io/v1alpha1
kind: OpenClawAgent
metadata:
  name: my-agent
  namespace: openclaw
spec:
  containerMode: immutable
  immutableProfile: python-ml
  
  identity:
    inline: |
      # My Agent Identity
      Role: Data Analyst
      Expertise: Statistical analysis, visualization
  
  memory:
    natsUrl: nats://nats.openclaw.svc:4222
    company:
      enabled: true
      mode: readonly
    group:
      enabled: true
      name: engineering
    individual:
      enabled: true
  
  resources:
    requests:
      cpu: "500m"
      memory: "1Gi"
```

## Project Structure

```
openclaw-platform/
├── api/v1alpha1/           # CRD type definitions
├── cmd/operator/           # Operator entrypoint
├── config/
│   ├── crd/               # CRD YAML manifests
│   ├── manager/           # Operator deployment
│   └── rbac/              # RBAC rules
├── deploy/
│   ├── images/            # Dockerfiles for agent profiles
│   └── nats/              # NATS deployment
├── examples/              # Example agent/swarm manifests
├── internal/controller/   # Operator reconciliation logic
└── pkg/memory/            # Memory client library
    ├── client/            # NATS KV client
    ├── company/           # Company-tier operations
    ├── group/             # Group-tier operations
    ├── swarm/             # Swarm coordination
    └── individual/        # Individual memory
```

## CRD Reference

### OpenClawAgent

```yaml
spec:
  containerMode: immutable | dynamic
  immutableProfile: minimal | python-ml | devops | web | data | security
  
  identity:
    configMap: name        # or inline: "..."
  soul:
    configMap: name        # or inline: "..."
  
  memory:
    natsUrl: nats://...
    company:
      enabled: true
      mode: readonly
    group:
      name: engineering
      mode: readwrite
    swarm:
      autoJoin: true
    individual:
      enabled: true
  
  swarm:
    enabled: true
    role: worker | orchestrator
    groups: [engineering, research]
  
  mutability:              # for dynamic mode
    enabled: true
    persistTools: true
    toolCachePVC: my-pvc
  
  capabilities:
    - network-access
    - kubectl
  
  resources:
    requests:
      cpu: "500m"
      memory: "1Gi"
```

### OpenClawSwarm

```yaml
spec:
  goal: "Review code for security vulnerabilities"
  orchestratorRef: orchestrator-agent-name
  workerSelector:
    matchLabels:
      platform.openclaw.io/swarm-enabled: "true"
  minWorkers: 2
  maxWorkers: 10
  timeout: 30m
  group: engineering
```

## Memory API

The memory client provides typed access to each tier:

```go
// Individual memory
individual := memory.NewIndividualManager(client, agentID)
individual.StoreIdentity(ctx, &Identity{...})
individual.UpdateContext(ctx, &WorkingContext{...})

// Group memory
group := memory.NewGroupManager(client, groupName, agentID)
group.AddKnowledge(ctx, &Knowledge{Topic: "go-patterns", ...})
group.AddLearning(ctx, &Learning{Title: "Retry pattern", ...})

// Swarm coordination
swarm := memory.NewSwarmManager(client, swarmID, agentID, RoleWorker)
swarm.JoinSwarm(ctx)
swarm.ReportProgress(ctx, shardID, 0.5, "Halfway done")
swarm.ShareDiscovery(ctx, &Discovery{Type: "security-issue", ...})

// Company memory
company := memory.NewCompanyManager(client, agentID)
tools, _ := company.GetToolsByProfile(ctx, "python-ml")
policies, _ := company.GetApplicablePolicies(ctx, agentID, group, profile)
```

## Development

```bash
# Run operator locally
make run

# Build operator image
make docker-build

# Build agent images
make docker-build-images

# Run tests
make test

# Create local Kind cluster
make kind-create
```

## Security Model

- Agents run as non-root (UID 1000)
- Read-only root filesystem
- Writable paths only via mounted volumes
- No privilege escalation
- Network policies restrict access to NATS + allowed endpoints
- Full trust between agents in same cluster (no mTLS overhead)

## License

Apache 2.0
