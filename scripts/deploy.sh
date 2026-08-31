#!/bin/bash
set -e

TAG="${1:-latest}"
DOCKER_USER="${DOCKER_USER:-YOUR_DOCKERHUB_USERNAME}"

echo "================================================================="
echo "Deploying Clinic Queue: Tag=[${TAG}], DockerHub User=[${DOCKER_USER}]"
echo "================================================================="

# Ensure shared podman network exists
podman network create clinic-net || true

# 1. Pull pre-built images from Docker Hub
echo "Pulling Docker images..."
podman pull docker.io/${DOCKER_USER}/clinic-backend:${TAG}
podman pull docker.io/${DOCKER_USER}/clinic-frontend:${TAG}

# 2. Restart Go Backend Container
echo "Stopping and removing previous Backend container..."
podman stop clinic-backend || true
podman rm clinic-backend || true

ENV_FLAG=""
if [ -f .env ]; then
  ENV_FLAG="--env-file .env"
fi

podman run -d --name clinic-backend \
  --restart always \
  --network clinic-net \
  -p 127.0.0.1:8080:8080 \
  ${ENV_FLAG} \
  docker.io/${DOCKER_USER}/clinic-backend:${TAG}

# 3. Restart Next.js Frontend Container
echo "Stopping and removing previous Frontend container..."
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
