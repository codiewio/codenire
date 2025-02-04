#!/bin/bash

set -e

docker system prune -f
echo "Docker prune finished"

# Main Sandbox
docker pull codiew/codenire-sandbox:latest

# Stop and Remove old container
docker ps -a --filter "name=sandbox_dev" -q | xargs docker stop || true
docker ps -a --filter "name=sandbox_dev" -q | xargs docker rm || true

docker network create --driver bridge --subnet=192.168.100.0/24 isolated_net || true

# start
docker run -d \
  --name isolated_gateway \
  --restart always \
  --network isolated_net \
  -p 3128:3128 \
  -v ./squid.conf:/etc/squid/squid.conf
  ubuntu/squid

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
  --isolated-network isolated_net \
  --isolated-gateway http://isolated_gateway:3128

# Show start logs
sleep 10
docker ps -q --filter "name=sandbox_dev" | xargs -I {}  docker logs --since 20s {}
