#!/bin/bash

set -e

# Main Playground
docker pull codiew/codenire-playground:latest

docker ps -a --filter "name=play_dev" -q | xargs docker stop || true
docker ps -a --filter "name=play_dev" -q | xargs docker rm || true

echo "Use $1 as sandbox backend"

docker run -d --name play_dev \
  --network host \
  --entrypoint "/playground" \
  --restart always \
  codiew/codenire-playground:latest \
  --backend-url "http://$1:80/run" \
  --port 8081

# Show start logs
docker ps -q --filter "name=play_dev" | xargs -I {}  docker logs --since 10s {}



#echo $DOCKER_REGISTRY_TOKEN | docker login --username foo --password-stdin