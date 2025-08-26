#!/bin/bash
set -ex

# --- Configuration ---
# The directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Source configuration from backend.env
if [ -f "$SCRIPT_DIR/backend.env" ]; then
    echo "Loading configuration from backend.env..."
    source "$SCRIPT_DIR/backend.env"
else
    echo "Error: backend.env file not found in $SCRIPT_DIR"
    echo "Please create backend.env based on backend.env.example"
    exit 1
fi

# Validate required variables
if [ -z "$ECR_REGISTRY_BACKEND_URI" ] || [ -z "$EC2_HOST" ] || [ -z "$EC2_USER" ] || [ -z "$STACK_NAME" ] || [ -z "$DOMAIN" ]; then
    echo "Error: Missing required configuration variables in backend.env"
    echo "Required: ECR_REGISTRY_BACKEND_URI, EC2_HOST, EC2_USER, STACK_NAME, DOMAIN"
    exit 1
fi

# --- Script ---
set -e # Exit on error
STACK_DIR="$SCRIPT_DIR"
REMOTE_STACK_DIR="/home/$EC2_USER/stacks/$STACK_NAME"
GIT_SHA=$(git rev-parse --short HEAD)

if [[ $(git diff --stat) != '' ]]; then
  GIT_SHA+="-dirty"
fi

IMAGE_TAG=pa11y-spike-$GIT_SHA

echo "--- Building and Pushing '$STACK_NAME' stack to $ECR_REGISTRY_BACKEND_URI with tag $IMAGE_TAG ---"

# 1. Login to ECR
aws ecr get-login-password --region eu-south-1 | docker login --username AWS --password-stdin $ECR_REGISTRY_BACKEND_URI

# docker_platform="--platform linux/amd64"
docker_platform="--platform linux/arm64"


pwd
# 2. Build the Docker image using buildx
docker buildx build $docker_platform -t "$ECR_REGISTRY_BACKEND_URI:$IMAGE_TAG" \
             --build-arg GIT_SHA=$GIT_SHA \
             -f "$ROOT_DIR/Dockerfile" "$ROOT_DIR" --push

echo "--- Deploying '$STACK_NAME' stack to $EC2_HOST ---"

# 4. Create the remote directory structure
echo "Creating remote directory: $REMOTE_STACK_DIR"
ssh "$EC2_USER@$EC2_HOST" "mkdir -p $REMOTE_STACK_DIR"

# 5. Copy the Docker Compose file to the remote host
echo "Copying docker-compose.yml..."
scp "$STACK_DIR/docker-compose.yml" "$EC2_USER@$EC2_HOST:$REMOTE_STACK_DIR/docker-compose.yml"

# 5b. Copy the environment file to the remote host
echo "Copying backend.env..."
scp "$STACK_DIR/backend.env" "$EC2_USER@$EC2_HOST:$REMOTE_STACK_DIR/backend.env"

# 6. Login to ECR, pull image, configure and start stack on the remote host
echo "--- Deploying on remote host ---"
ssh "$EC2_USER@$EC2_HOST" "
  set -e
  echo '--> Logging into ECR...'
  aws ecr get-login-password --region eu-south-1 | docker login --username AWS --password-stdin $ECR_REGISTRY_BACKEND_URI

  echo '--> Pulling latest image...'
  docker pull $ECR_REGISTRY_BACKEND_URI:$IMAGE_TAG

  cd $REMOTE_STACK_DIR

  echo '--> Updating docker-compose.yml...'
  # The docker-compose.yml file should use __ECR_REGISTRY_URI__, __IMAGE_TAG__, and __DOMAIN__ as placeholders.
  TIMESTAMP=\"# Deployed at: $(date) - Git SHA: $GIT_SHA\"

  # Use sed to replace the placeholders.
  # Using '|' as a separator for sed to avoid conflicts with slashes in the URI.
  sed -i \
    -e \"s|__ECR_REGISTRY_URI__|$ECR_REGISTRY_BACKEND_URI|g\" \
    -e \"s|__IMAGE_TAG__|$IMAGE_TAG|g\" \
    -e \"s|__DOMAIN__|$DOMAIN|g\" \
    docker-compose.yml

  echo \"$TIMESTAMP\" >> docker-compose.yml

  echo '--> Starting the stack...'
  docker compose up -d
"

echo "--- Deployment of '$STACK_NAME' complete ---"
