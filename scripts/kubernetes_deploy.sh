#!/bin/bash

# Define Docker image and Kubernetes namespace
IMAGE_NAME="eth-proxy_eth-proxy"
DOCKER_USERNAME="luishsoares"
IMAGE_TAG="latest"
K8S_DEPLOYMENT_NAME="eth-proxy-deployment"
K8S_SERVICE_NAME="eth-proxy-service"
K8S_INGRESS_NAME="eth-proxy-ingress"
K8S_NAMESPACE="default"
K8S_CONTEXT="do-ams3-k8s-1-29-1-do-0-ams3-1708852461132"

# Build the Docker image using Docker Compose
docker-compose build eth-proxy

# Tag the built image
docker tag $IMAGE_NAME:latest $DOCKER_USERNAME/$IMAGE_NAME:$IMAGE_TAG

# Push the image to Docker Hub
docker push $DOCKER_USERNAME/$IMAGE_NAME:$IMAGE_TAG

kubectl config use-context $K8S_CONTEXT

# Remove existing Kubernetes resources if they exist
kubectl delete deployment $K8S_DEPLOYMENT_NAME --ignore-not-found=true -n $K8S_NAMESPACE
kubectl delete service $K8S_SERVICE_NAME --ignore-not-found=true -n $K8S_NAMESPACE
kubectl delete ingress $K8S_INGRESS_NAME --ignore-not-found=true -n $K8S_NAMESPACE

# Wait a bit for resources to be cleaned up (optional, adjust as necessary)
sleep 10

# Apply the Kubernetes deployment, service, ingress, and probes manifest for eth-proxy
kubectl apply -f ../deployments/eth-proxy-deployment.yaml
kubectl apply -f ../deployments/eth-proxy-service.yaml
kubectl apply -f ../deployments/eth-proxy-ingress.yaml
kubectl apply -f ../deployments/eth-proxy-probes.yaml

# API Keys
kubectl apply -f ../utils/api-secret.yaml
kubectl apply -f ../utils/key-auth-plugin.yaml

# Update the eth-proxy deployment to use the latest image
kubectl set image deployment/$K8S_DEPLOYMENT_NAME eth-proxy-container=$DOCKER_USERNAME/$IMAGE_NAME:$IMAGE_TAG -n $K8S_NAMESPACE

# Ensure at least 3 replicas for eth-proxy
kubectl scale deployment $K8S_DEPLOYMENT_NAME --replicas=3 -n $K8S_NAMESPACE

echo "Deployment updated and scaled successfully."
