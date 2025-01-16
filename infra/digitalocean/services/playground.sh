#!/bin/bash

set -e

# Main Playground
docker pull codiew/codenire-playground:latest

docker ps -a --filter "name=play_dev" -q | xargs docker stop || true
docker ps -a --filter "name=play_dev" -q | xargs docker rm || true
docker ps -a --filter "name=traefik" -q | xargs docker stop || true
docker ps -a --filter "name=traefik" -q | xargs docker rm || true

docker run -d --name play_dev \
  -p 80:8081 \
  --add-host=sandbox-host:"$1" \
  --entrypoint "/playground" \
  --restart always \
  \
  codiew/codenire-playground:latest \
  \
  --backend-url "http://sandbox-host:80/run" \
  --port 8081

# Show start logs
sleep 3
docker ps -q --filter "name=play_dev" | xargs -I {}  docker logs --since 20s {}

