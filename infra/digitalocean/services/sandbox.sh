#!/bin/bash

set -e

# Main Sandbox
docker pull codiew/codenire-sandbox:latest

# Stop and Remove old container
docker ps -a --filter "name=sandbox_dev" -q | xargs docker stop || true
docker ps -a --filter "name=sandbox_dev" -q | xargs docker rm || true

# TODO:: Prepare /ops/dockerfiles

# Start app
docker run -d --name sandbox_dev \
  -p 8080:80/tcp \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /ops/dockerfiles:/dockerfiles \
  --restart=always \
  --entrypoint "/usr/local/bin/sandbox" \
  codiew/codenire-sandbox:latest \
  --dockerFilesPath /dockerfiles \
  --replicaContainerCnt 1

# Show start logs
docker ps -q --filter "name=sandbox_dev" | xargs -I {}  docker logs --since 10s {}
