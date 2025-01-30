#!/bin/bash

set -e

docker system prune -f
echo "Docker prune finished"

# Main Sandbox
docker pull codiew/codenire-sandbox:latest

# Stop and Remove old container
docker ps -a --filter "name=sandbox_dev" -q | xargs docker stop || true
docker ps -a --filter "name=sandbox_dev" -q | xargs docker rm || true

# Start app
docker run -d --name sandbox_dev \
  -p 80:8081 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --restart=always \
  --entrypoint "/usr/local/bin/sandbox" \
  codiew/codenire-sandbox:latest \
  --dockerFilesPath /dockerfiles \
  --replicaContainerCnt 1 \
  --port 8081

# Show start logs
sleep 10
docker ps -q --filter "name=sandbox_dev" | xargs -I {}  docker logs --since 20s {}
