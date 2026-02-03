#!/bin/bash
# Initialize OpenClaw memory buckets in NATS JetStream

set -e

NATS_URL="${NATS_URL:-nats://nats.openclaw.svc:4222}"

echo "Connecting to NATS at $NATS_URL..."

# Wait for NATS to be ready
until nats server check connection --server="$NATS_URL" 2>/dev/null; do
  echo "Waiting for NATS to be ready..."
  sleep 2
done

echo "NATS is ready. Creating memory buckets..."

# Company-level buckets (permanent, high replication)
echo "Creating company-level buckets..."

nats kv add memory.company.knowledge \
  --server="$NATS_URL" \
  --description="Company-wide knowledge base" \
  --history=25 \
  --replicas=3 \
  --max-bucket-size=1GB \
  2>/dev/null || echo "Bucket memory.company.knowledge already exists"

nats kv add memory.company.tools \
  --server="$NATS_URL" \
  --description="Company tool catalog" \
  --history=10 \
  --replicas=3 \
  --max-bucket-size=100MB \
  2>/dev/null || echo "Bucket memory.company.tools already exists"

nats kv add memory.company.policies \
  --server="$NATS_URL" \
  --description="Company policies and rules" \
  --history=50 \
  --replicas=3 \
  --max-bucket-size=100MB \
  2>/dev/null || echo "Bucket memory.company.policies already exists"

# Create default groups
echo "Creating default group buckets..."

for group in engineering management quant research product design; do
  nats kv add "memory.group.$group" \
    --server="$NATS_URL" \
    --description="Group memory: $group" \
    --history=15 \
    --replicas=3 \
    --max-bucket-size=500MB \
    2>/dev/null || echo "Bucket memory.group.$group already exists"
done

# Create streams for coordination signals
echo "Creating coordination streams..."

nats stream add openclaw-swarm-signals \
  --server="$NATS_URL" \
  --subjects="openclaw.swarm.*.signals" \
  --description="Swarm coordination signals" \
  --retention=limits \
  --max-age=24h \
  --max-msgs=100000 \
  --max-bytes=100MB \
  --replicas=1 \
  --discard=old \
  2>/dev/null || echo "Stream openclaw-swarm-signals already exists"

nats stream add openclaw-events \
  --server="$NATS_URL" \
  --subjects="openclaw.events.>" \
  --description="OpenClaw platform events" \
  --retention=limits \
  --max-age=7d \
  --max-msgs=1000000 \
  --max-bytes=1GB \
  --replicas=3 \
  --discard=old \
  2>/dev/null || echo "Stream openclaw-events already exists"

echo "Bucket initialization complete!"

# List all buckets
echo ""
echo "Current KV buckets:"
nats kv ls --server="$NATS_URL"

echo ""
echo "Current streams:"
nats stream ls --server="$NATS_URL"
