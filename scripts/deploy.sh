#!/bin/bash
set -e

TAG="${1:-latest}"
DOCKER_USER="${DOCKER_USER:-YOUR_DOCKERHUB_USERNAME}"

echo "================================================================="
echo "Deploying Clinic Queue: Tag=[${TAG}], DockerHub User=[${DOCKER_USER}]"
echo "================================================================="

# Ensure shared podman network exists
podman network create clinic-net || true

# Load .env file from /opt/clinic-queue/.env if present
if [ -f .env ]; then
  echo "Loading environment variables from .env"
  export $(grep -v '^#' .env | xargs)
fi

# 1. Pull pre-built images from Docker Hub
echo "Pulling Docker images..."
podman pull docker.io/${DOCKER_USER}/clinic-backend:${TAG}
podman pull docker.io/${DOCKER_USER}/clinic-frontend:${TAG}

# 2. Restart Go Backend Container
echo "Restarting Go Backend container..."
podman stop clinic-backend || true
podman rm clinic-backend || true

podman run -d --name clinic-backend \
  --restart always \
  --network clinic-net \
  -p 127.0.0.1:8080:8080 \
  -e PORT="${PORT:-8080}" \
  -e DATABASE_URL \
  -e JWT_SECRET \
  -e NATS_URL \
  -e CASBIN_MODEL_PATH="${CASBIN_MODEL_PATH:-config/rbac_model.conf}" \
  -e CASBIN_POLICY_PATH="${CASBIN_POLICY_PATH:-config/rbac_policy.csv}" \
  docker.io/${DOCKER_USER}/clinic-backend:${TAG}

# 3. Restart Next.js Frontend Container
echo "Restarting Next.js Frontend container..."
podman stop clinic-frontend || true
podman rm clinic-frontend || true

podman run -d --name clinic-frontend \
  --restart always \
  --network clinic-net \
  -p 127.0.0.1:3000:3000 \
  docker.io/${DOCKER_USER}/clinic-frontend:${TAG}

echo "================================================================="
echo "Deployment SUCCESSFUL for tag: ${TAG}"
echo "================================================================="
podman ps
