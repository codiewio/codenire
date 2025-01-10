#!/bin/bash

set -e

# Main Sandbox
docker pull codiew/codenire-sandbox:latest

# copy dockerfiles for sandbox in tmp dir from var.dockerfiles_git_repo (terraform variable)
tmp_dir=$(mktemp -d)
cd "$tmp_dir" && git clone "$1" .


# Stop and Remove old container
docker ps -a --filter "name=sandbox_dev" -q | xargs docker stop || true
docker ps -a --filter "name=sandbox_dev" -q | xargs docker rm || true

# replace dockerfiles
rm -rf /ops/dockerfiles/*
cp -r "$tmp_dir"/* /ops/dockerfiles/

echo "Used dockefiles configs:"
ls -la /ops/dockerfiles

# Start app
docker run -d --name sandbox_dev \
  -p 80:8081 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /ops/dockerfiles:/dockerfiles \
  --restart=always \
  --entrypoint "/usr/local/bin/sandbox" \
  codiew/codenire-sandbox:latest \
  --dockerFilesPath /dockerfiles \
  --replicaContainerCnt 1 \
  --port 8081

# Show start logs
docker ps -q --filter "name=sandbox_dev" | xargs -I {}  docker logs --since 10s {}
