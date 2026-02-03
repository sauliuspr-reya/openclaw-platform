#!/bin/bash
# Setup script for GCP Artifact Registry and GitHub Actions authentication
# 
# Usage: ./hack/setup-gcp-github-auth.sh [--github-repo OWNER/REPO]
#
# Prerequisites:
#   - gcloud CLI installed and authenticated
#   - gh CLI installed and authenticated (for setting GitHub secrets)

set -e

# Configuration
PROJECT_ID="mainnet-473609"
REGION="europe-west3"
REPOSITORY="reya"
REGISTRY="${REGION}-docker.pkg.dev"
SA_NAME="github-actions-openclaw"
SA_EMAIL="${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"
WIF_POOL="github-actions"
WIF_PROVIDER="github"

# Parse arguments
GITHUB_REPO=""
while [[ $# -gt 0 ]]; do
  case $1 in
    --github-repo)
      GITHUB_REPO="$2"
      shift 2
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

echo "================================================"
echo "OpenClaw GCP Artifact Registry & GitHub Auth Setup"
echo "================================================"
echo ""
echo "Project:    ${PROJECT_ID}"
echo "Region:     ${REGION}"
echo "Repository: ${REPOSITORY}"
echo "Registry:   ${REGISTRY}/${PROJECT_ID}/${REPOSITORY}"
echo ""

# Get project number
echo "📋 Getting project number..."
PROJECT_NUMBER=$(gcloud projects describe ${PROJECT_ID} --format='value(projectNumber)')
echo "   Project number: ${PROJECT_NUMBER}"
echo ""

# Step 1: Verify/List Artifact Registry repositories
echo "📦 Step 1: Checking Artifact Registry repositories..."
echo ""
echo "   Listing repositories in ${REGION}:"
gcloud artifacts repositories list \
  --project="${PROJECT_ID}" \
  --location="${REGION}" \
  --format="table(name,format,createTime)" 2>/dev/null || echo "   No repositories found or insufficient permissions"
echo ""

# Check if repository exists
if gcloud artifacts repositories describe ${REPOSITORY} \
  --project="${PROJECT_ID}" \
  --location="${REGION}" &>/dev/null; then
  echo "   ✅ Repository '${REPOSITORY}' exists"
else
  echo "   ⚠️  Repository '${REPOSITORY}' not found"
  echo ""
  read -p "   Create repository? (y/n) " -n 1 -r
  echo ""
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "   Creating repository..."
    gcloud artifacts repositories create ${REPOSITORY} \
      --project="${PROJECT_ID}" \
      --location="${REGION}" \
      --repository-format=docker \
      --description="OpenClaw container images"
    echo "   ✅ Repository created"
  fi
fi
echo ""

# Step 2: List existing images in the repository
echo "🖼️  Step 2: Listing existing images in repository..."
gcloud artifacts docker images list \
  "${REGISTRY}/${PROJECT_ID}/${REPOSITORY}" \
  --include-tags \
  --format="table(package,tags,createTime)" \
  --limit=20 2>/dev/null || echo "   No images found or repository is empty"
echo ""

# Step 3: Enable required APIs
echo "🔧 Step 3: Enabling required APIs..."
gcloud services enable \
  iamcredentials.googleapis.com \
  artifactregistry.googleapis.com \
  --project="${PROJECT_ID}"
echo "   ✅ APIs enabled"
echo ""

# Step 4: Create/verify Workload Identity Pool
echo "🔐 Step 4: Setting up Workload Identity Federation..."
echo ""

# Check if pool exists
if gcloud iam workload-identity-pools describe ${WIF_POOL} \
  --project="${PROJECT_ID}" \
  --location="global" &>/dev/null; then
  echo "   ✅ Workload Identity Pool '${WIF_POOL}' exists"
else
  echo "   Creating Workload Identity Pool..."
  gcloud iam workload-identity-pools create ${WIF_POOL} \
    --project="${PROJECT_ID}" \
    --location="global" \
    --display-name="GitHub Actions Pool"
  echo "   ✅ Pool created"
fi

# Check if provider exists
if gcloud iam workload-identity-pools providers describe ${WIF_PROVIDER} \
  --project="${PROJECT_ID}" \
  --location="global" \
  --workload-identity-pool=${WIF_POOL} &>/dev/null; then
  echo "   ✅ Workload Identity Provider '${WIF_PROVIDER}' exists"
else
  echo "   Creating Workload Identity Provider..."
  gcloud iam workload-identity-pools providers create-oidc ${WIF_PROVIDER} \
    --project="${PROJECT_ID}" \
    --location="global" \
    --workload-identity-pool=${WIF_POOL} \
    --display-name="GitHub OIDC" \
    --attribute-mapping="google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.repository=assertion.repository,attribute.repository_owner=assertion.repository_owner" \
    --attribute-condition="assertion.repository_owner == 'reya-labs'" \
    --issuer-uri="https://token.actions.githubusercontent.com"
  echo "   ✅ Provider created"
fi
echo ""

# Step 5: Create/verify Service Account
echo "👤 Step 5: Setting up Service Account..."
if gcloud iam service-accounts describe ${SA_EMAIL} \
  --project="${PROJECT_ID}" &>/dev/null; then
  echo "   ✅ Service Account '${SA_NAME}' exists"
else
  echo "   Creating Service Account..."
  gcloud iam service-accounts create ${SA_NAME} \
    --project="${PROJECT_ID}" \
    --display-name="GitHub Actions OpenClaw"
  echo "   ✅ Service Account created"
fi
echo ""

# Step 6: Grant Artifact Registry permissions
echo "🔑 Step 6: Granting Artifact Registry permissions..."
gcloud artifacts repositories add-iam-policy-binding ${REPOSITORY} \
  --project="${PROJECT_ID}" \
  --location="${REGION}" \
  --member="serviceAccount:${SA_EMAIL}" \
  --role="roles/artifactregistry.writer" \
  --quiet

gcloud artifacts repositories add-iam-policy-binding ${REPOSITORY} \
  --project="${PROJECT_ID}" \
  --location="${REGION}" \
  --member="serviceAccount:${SA_EMAIL}" \
  --role="roles/artifactregistry.reader" \
  --quiet
echo "   ✅ Permissions granted"
echo ""

# Step 7: Bind Workload Identity to Service Account
echo "🔗 Step 7: Binding Workload Identity to Service Account..."
if [ -n "${GITHUB_REPO}" ]; then
  MEMBER="principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${WIF_POOL}/attribute.repository/${GITHUB_REPO}"
  
  gcloud iam service-accounts add-iam-policy-binding ${SA_EMAIL} \
    --project="${PROJECT_ID}" \
    --role="roles/iam.workloadIdentityUser" \
    --member="${MEMBER}" \
    --quiet
  echo "   ✅ Bound to repository: ${GITHUB_REPO}"
else
  echo "   ⚠️  No GitHub repo specified. Run with --github-repo OWNER/REPO to bind."
  echo "   Manual command:"
  echo ""
  echo "   gcloud iam service-accounts add-iam-policy-binding ${SA_EMAIL} \\"
  echo "     --project=\"${PROJECT_ID}\" \\"
  echo "     --role=\"roles/iam.workloadIdentityUser\" \\"
  echo "     --member=\"principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${WIF_POOL}/attribute.repository/YOUR_ORG/YOUR_REPO\""
fi
echo ""

# Step 8: Generate GitHub Secrets
echo "================================================"
echo "📝 GitHub Secrets Configuration"
echo "================================================"
echo ""

WIF_PROVIDER_FULL="projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${WIF_POOL}/providers/${WIF_PROVIDER}"

echo "Secret: GCP_WORKLOAD_IDENTITY_PROVIDER"
echo "Value:  ${WIF_PROVIDER_FULL}"
echo ""
echo "Secret: GCP_SERVICE_ACCOUNT"
echo "Value:  ${SA_EMAIL}"
echo ""

# Set GitHub secrets if gh CLI is available and repo is specified
if [ -n "${GITHUB_REPO}" ] && command -v gh &>/dev/null; then
  echo ""
  read -p "Set GitHub secrets now using gh CLI? (y/n) " -n 1 -r
  echo ""
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Setting GitHub secrets..."
    gh secret set GCP_WORKLOAD_IDENTITY_PROVIDER --repo "${GITHUB_REPO}" --body "${WIF_PROVIDER_FULL}"
    gh secret set GCP_SERVICE_ACCOUNT --repo "${GITHUB_REPO}" --body "${SA_EMAIL}"
    echo "   ✅ GitHub secrets configured!"
  fi
fi

echo ""
echo "================================================"
echo "✅ Setup Complete!"
echo "================================================"
echo ""
echo "Registry URL: ${REGISTRY}/${PROJECT_ID}/${REPOSITORY}/"
echo ""
echo "Example image paths:"
echo "  - ${REGISTRY}/${PROJECT_ID}/${REPOSITORY}/openclaw-operator:latest"
echo "  - ${REGISTRY}/${PROJECT_ID}/${REPOSITORY}/openclaw-agent-python-ml:latest"
echo "  - ${REGISTRY}/${PROJECT_ID}/${REPOSITORY}/hephaestus:latest"
echo ""
echo "To push an image manually:"
echo "  gcloud auth configure-docker ${REGISTRY}"
echo "  docker push ${REGISTRY}/${PROJECT_ID}/${REPOSITORY}/IMAGE:TAG"
echo ""
