#!/bin/bash

set -e

# Main Playground
docker pull codiew/codenire-playground:latest

docker ps -a --filter "name=play_dev" -q | xargs docker stop
docker ps -a --filter "name=play_dev" -q | xargs docker rm

echo "Use $1 as backend"

docker run -d --name play_dev \
  -p 80:80 \
  --entrypoint "/app/codenire" \
  --restart always \
  codiew/codenire-playground:latest \
  --backend-url "$1"/run \
  --port 80