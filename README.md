## The Ethereum Proxy Service

The Ethereum Proxy Service is designed to provide a highly available and scalable solution for interacting with Ethereum nodes in a NON-PRODUCTION environment. It handles inconsistent data returned by a set of clients and distributes load across multiple Ethereum nodes.

![AWS Deployment](https://github.com/luishsr/eth-proxy/assets/80909424/33c9af93-3d38-42f5-98c1-93935a83f759)

## Features

-   High Availability: Ensures continuous service operation by distributing requests across multiple Ethereum nodes.
-   Load Balancing: Dynamically balances load to maintain performance and minimize latency.
-   Inconsistent Data Handling: Implements logic to manage and mitigate inconsistencies in data returned by different Ethereum nodes.
-   Prometheus Metrics: Exposes a /metrics endpoint for monitoring service performance and health with Prometheus.
-   Health Checks: Includes /healthz and /ready HTTP endpoints for liveness and readiness probes, facilitating smooth integration with Kubernetes or other orchestration tools.
-   Scalability: Designed to scale horizontally, allowing for an increased number of clients and requests without compromising on performance.

## Getting Started

## Prerequisites

-   Docker and Docker Compose for building and running the service locally.
-   Kubernetes cluster for deploying the service in a test environment.
-   Kong API service deployed on Kubernetes
-   Prometheus setup for monitoring (optional).

## Local Setup

Clone the repository:

    git clone https://github.com/luishsr/eth-proxy cd <repository-name>

Build the Docker image:

    docker-compose build eth-proxy

Run the service locally:

    docker-compose up

Deployment on Kubernetes

Make sure your kubectl is configured to interact with your Kubernetes cluster.

Deploy the service:

Run the provided deployment script to deploy the Ethereum Proxy Service to your Kubernetes cluster:

    ./kubernetes_deploy.sh

This script will remove any existing deployments and services before deploying the new configuration.

## Accessing the Service

-   The Ethereum Proxy Service will be accessible at the IP address assigned by your Kubernetes cluster or Docker setup.
-   Access Prometheus metrics at /metrics.
-   Check the service's health and readiness at the /healthz and /ready endpoints, respectively.

## Monitoring with Prometheus

To monitor the Ethereum Proxy Service with Prometheus:

Ensure Prometheus is running in your environment. Configure Prometheus to scrape metrics from the service's /metrics endpoint.

## Contributing

Contributions are welcome! Please feel free to submit pull requests or create issues for bugs, questions, and feature requests.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
