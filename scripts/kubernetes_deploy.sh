#!/bin/bash

# Define Docker image and Kubernetes namespace
IMAGE_NAME="eth-proxy_eth-proxy"
DOCKER_USERNAME="luishsoares"
IMAGE_TAG="latest"
K8S_DEPLOYMENT_NAME="eth-proxy-deployment"
K8S_NAMESPACE="default"

# Function to stop any existing port-forward processes for specified local ports
stop_port_forwarding() {
    # List of local ports used for port-forwarding
    local_ports=("$@")
    for port in "${local_ports[@]}"; do
        # Find processes listening on the given port and terminate them
        lsof -ti:$port -sTCP:LISTEN | xargs -r kill
    done
}

# Step 1: Build the Docker image using Docker Compose
docker-compose build eth-proxy

# Step 2: Tag the built image
docker tag $IMAGE_NAME:latest $DOCKER_USERNAME/$IMAGE_NAME:$IMAGE_TAG

# Step 3: Push the image to Docker Hub
docker push $DOCKER_USERNAME/$IMAGE_NAME:$IMAGE_TAG

kubectl config use-context local

# Step 4: Apply the Kubernetes deployment and service manifest for eth-proxy
kubectl apply -f ../deployments/eth-proxy-deployment.yaml
kubectl apply -f ../deployments/eth-proxy-service.yaml
kubectl apply -f ../deployments/eth-proxy-loadbalancer.yaml

# Step 5: Update the eth-proxy deployment to use the latest image
kubectl set image deployment/$K8S_DEPLOYMENT_NAME eth-proxy-container=$DOCKER_USERNAME/$IMAGE_NAME:$IMAGE_TAG -n $K8S_NAMESPACE

# Step 6: Ensure at least 3 replicas for eth-proxy
kubectl scale deployment $K8S_DEPLOYMENT_NAME --replicas=3 -n $K8S_NAMESPACE

# Port-forwarding
kubectl port-forward svc/eth-loadbalancer-service 8080:80 -n $K8S_NAMESPACE

echo "Deployment updated and scaled successfully."