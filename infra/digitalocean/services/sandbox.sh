#!/bin/bash

set -e

docker system prune -f
echo "Docker prune finished"

# Main Sandbox
docker pull codiew/codenire-sandbox:latest

# Stop and Remove old resources
docker ps -a --filter "name=sandbox_dev" --filter "name=isolated_gateway" -q | xargs docker rm -f || true
docker network create --driver bridge --subnet=192.168.100.0/24 isolated_net || true

# start
docker run -d \
  --name isolated_gateway \
  --restart always \
  --network isolated_net \
  -p 3128:3128 \
  codiew/codenire-deproxy:latest

# Start app
docker run -d --name sandbox_dev \
  --network=isolated_net \
  -p 80:8081 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --restart=always \
  --entrypoint "/usr/local/bin/sandbox" \
  codiew/codenire-sandbox:latest \
  --dockerFilesPath /dockerfiles \
  --replicaContainerCnt 1 \
  --port 8081 \
  --isolatedNetwork isolated_net \
  --isolatedGateway http://isolated_gateway:3128

# Show start logs
sleep 10
docker ps -q --filter "name=sandbox_dev" | xargs -I {}  docker logs --since 20s {}
