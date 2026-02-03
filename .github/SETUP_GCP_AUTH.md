# Setting up GCP Authentication for GitHub Actions

This document explains how to configure Workload Identity Federation for secure authentication between GitHub Actions and GCP Artifact Registry.

## Overview

We use **Workload Identity Federation** instead of service account keys for security:
- No long-lived credentials to rotate or leak
- GitHub Actions authenticates directly with GCP using OIDC tokens
- Fine-grained access control per repository

## Prerequisites

- GCP project: `mainnet-473609`
- Artifact Registry repository: `europe-west3-docker.pkg.dev/mainnet-473609/reya`
- `gcloud` CLI installed and authenticated

## Setup Steps

### 1. Enable Required APIs

```bash
gcloud services enable \
  iamcredentials.googleapis.com \
  artifactregistry.googleapis.com \
  --project=mainnet-473609
```

### 2. Create Workload Identity Pool

```bash
gcloud iam workload-identity-pools create "github-actions" \
  --project="mainnet-473609" \
  --location="global" \
  --display-name="GitHub Actions Pool"
```

### 3. Create Workload Identity Provider

```bash
gcloud iam workload-identity-pools providers create-oidc "github" \
  --project="mainnet-473609" \
  --location="global" \
  --workload-identity-pool="github-actions" \
  --display-name="GitHub" \
  --attribute-mapping="google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.repository=assertion.repository,attribute.repository_owner=assertion.repository_owner" \
  --issuer-uri="https://token.actions.githubusercontent.com"
```

### 4. Create Service Account

```bash
gcloud iam service-accounts create "github-actions-openclaw" \
  --project="mainnet-473609" \
  --display-name="GitHub Actions OpenClaw"
```

### 5. Grant Artifact Registry Permissions

```bash
# Grant write access to Artifact Registry
gcloud artifacts repositories add-iam-policy-binding reya \
  --project="mainnet-473609" \
  --location="europe-west3" \
  --member="serviceAccount:github-actions-openclaw@mainnet-473609.iam.gserviceaccount.com" \
  --role="roles/artifactregistry.writer"

# Also grant reader for pulling base images during builds
gcloud artifacts repositories add-iam-policy-binding reya \
  --project="mainnet-473609" \
  --location="europe-west3" \
  --member="serviceAccount:github-actions-openclaw@mainnet-473609.iam.gserviceaccount.com" \
  --role="roles/artifactregistry.reader"
```

### 6. Allow GitHub Actions to Impersonate Service Account

Replace `YOUR_ORG/YOUR_REPO` with your actual GitHub organization and repository:

```bash
# For a specific repository
gcloud iam service-accounts add-iam-policy-binding \
  "github-actions-openclaw@mainnet-473609.iam.gserviceaccount.com" \
  --project="mainnet-473609" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/github-actions/attribute.repository/YOUR_ORG/openclaw-platform"
```

To get your project number:
```bash
gcloud projects describe mainnet-473609 --format='value(projectNumber)'
```

### 7. Configure GitHub Secrets

#### Option A: Generate and set secrets using GitHub CLI (recommended)

```bash
# Set your GitHub repository
GITHUB_REPO="reya-labs/openclaw-platform"

# Get project number
PROJECT_NUMBER=$(gcloud projects describe mainnet-473609 --format='value(projectNumber)')

# Generate the workload identity provider value
WIF_PROVIDER="projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/github-actions/providers/github"

# Service account email
SERVICE_ACCOUNT="github-actions-openclaw@mainnet-473609.iam.gserviceaccount.com"

# Set GitHub secrets using gh CLI
gh secret set GCP_WORKLOAD_IDENTITY_PROVIDER --repo "${GITHUB_REPO}" --body "${WIF_PROVIDER}"
gh secret set GCP_SERVICE_ACCOUNT --repo "${GITHUB_REPO}" --body "${SERVICE_ACCOUNT}"

echo "✅ GitHub secrets configured!"
echo "   GCP_WORKLOAD_IDENTITY_PROVIDER: ${WIF_PROVIDER}"
echo "   GCP_SERVICE_ACCOUNT: ${SERVICE_ACCOUNT}"
```

#### Option B: Get values and set manually via GitHub UI

```bash
# Get project number
PROJECT_NUMBER=$(gcloud projects describe mainnet-473609 --format='value(projectNumber)')

# Print the values to copy into GitHub UI
echo ""
echo "======================================"
echo "GitHub Secrets to Configure"
echo "======================================"
echo ""
echo "Secret: GCP_WORKLOAD_IDENTITY_PROVIDER"
echo "Value:  projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/github-actions/providers/github"
echo ""
echo "Secret: GCP_SERVICE_ACCOUNT"
echo "Value:  github-actions-openclaw@mainnet-473609.iam.gserviceaccount.com"
echo ""
echo "======================================"
echo "Go to: https://github.com/YOUR_ORG/openclaw-platform/settings/secrets/actions"
echo "======================================"
```

| Secret Name | Value |
|-------------|-------|
| `GCP_WORKLOAD_IDENTITY_PROVIDER` | `projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/github-actions/providers/github` |
| `GCP_SERVICE_ACCOUNT` | `github-actions-openclaw@mainnet-473609.iam.gserviceaccount.com` |

## Verification

After setup, you can verify the configuration:

```bash
# List workload identity pools
gcloud iam workload-identity-pools list \
  --project="mainnet-473609" \
  --location="global"

# List providers in the pool
gcloud iam workload-identity-pools providers list \
  --project="mainnet-473609" \
  --location="global" \
  --workload-identity-pool="github-actions"

# Check service account IAM policy
gcloud iam service-accounts get-iam-policy \
  "github-actions-openclaw@mainnet-473609.iam.gserviceaccount.com" \
  --project="mainnet-473609"
```

## Troubleshooting

### "Permission denied" errors

1. Verify the service account has Artifact Registry Writer role
2. Check the workload identity binding matches your repository exactly
3. Ensure the workflow has `id-token: write` permission

### "Invalid JWT" errors

1. Verify the issuer URI is exactly `https://token.actions.githubusercontent.com`
2. Check attribute mappings are correct

### Images not pushing

1. Verify Docker is configured: `gcloud auth configure-docker europe-west3-docker.pkg.dev`
2. Check the image tag format matches the registry path

## Security Notes

- The service account only has access to push images to the `reya` repository
- Access is scoped to the specific GitHub repository
- No service account keys are used - authentication is ephemeral
- Audit logs are available in GCP Cloud Audit Logs
