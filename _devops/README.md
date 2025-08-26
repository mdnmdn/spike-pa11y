# DevOps Deployment Scripts

This directory contains the deployment scripts and configuration files for the a11y-buddy backend service.

## Files Overview

- `deploy-backend.sh` - Main deployment script that builds, pushes, and deploys the backend service
- `docker-compose.yml` - Docker Compose configuration with placeholders for dynamic values
- `backend.env` - Environment configuration file (excluded from git)
- `backend.env.example` - Example configuration file showing required variables
- `pa11y.json` - Pa11y configuration file
- `Dockerfile` - Docker image definition (located in project root)

## Setup

1. **Create your configuration file:**
   ```bash
   cp backend.env.example backend.env
   ```

2. **Edit `backend.env` with your actual values:**
   ```bash
   # Server Configuration
   DOMAIN=your-actual-domain.com
   
   ECR_REGISTRY_BACKEND_URI="123456789012.dkr.ecr.your-region.amazonaws.com/your-repo-name"
   EC2_HOST="your-ec2-host.com"
   EC2_USER="app-user"
   STACK_NAME="backend"
   ```

## Configuration Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DOMAIN` | The domain where your service will be accessible via Traefik | `api.example.com` |
| `ECR_REGISTRY_BACKEND_URI` | AWS ECR repository URI for the backend image | `123456789012.dkr.ecr.us-east-1.amazonaws.com/my-backend` |
| `EC2_HOST` | The hostname or IP of your EC2 deployment target | `ec2-host.example.com` |
| `EC2_USER` | SSH user for connecting to the EC2 instance | `app-user` |
| `STACK_NAME` | Name of the deployment stack | `backend` |

## Deployment Process

The `deploy-backend.sh` script performs the following steps:

1. **Configuration Loading**: Sources variables from `backend.env`
2. **Validation**: Ensures all required variables are set
3. **Docker Build**: Builds the Docker image using the project's Dockerfile
4. **ECR Push**: Pushes the image to AWS ECR with a git-based tag
5. **Remote Deployment**: 
   - Creates necessary directories on the target EC2 instance
   - Copies docker-compose.yml and backend.env to the remote host
   - Logs into ECR on the remote host
   - Pulls the latest image
   - Updates docker-compose.yml with actual values (replaces placeholders)
   - Starts the service using Docker Compose

## Usage

1. **Ensure you have the required tools:**
   - Docker with buildx support
   - AWS CLI configured with ECR access
   - SSH access to your EC2 instance

2. **Run the deployment:**
   ```bash
   ./deploy-backend.sh
   ```

## Docker Compose Placeholders

The `docker-compose.yml` file uses the following placeholders that are replaced during deployment:

- `__ECR_REGISTRY_URI__` - Replaced with `ECR_REGISTRY_BACKEND_URI`
- `__IMAGE_TAG__` - Replaced with the git SHA (and "-dirty" suffix if uncommitted changes exist)
- `__DOMAIN__` - Replaced with `DOMAIN` for Traefik routing

## Prerequisites

- AWS ECR repository created and accessible
- EC2 instance with Docker and Docker Compose installed
- Traefik reverse proxy running on the EC2 instance with the "web" network
- SSH key-based authentication configured for the EC2 instance
- AWS CLI configured with appropriate permissions

## Security Notes

- The `backend.env` file is excluded from git to prevent sensitive information from being committed
- Always use the example file as a template and never commit actual credentials
- Ensure your EC2 instance has proper security groups and access controls
