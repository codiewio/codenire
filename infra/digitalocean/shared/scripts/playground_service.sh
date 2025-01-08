#!/bin/bash

set -e

cd /app

# --- Specific plugin ---
# TODO:: make tmp dir and build to ./var/artefacts/plugins
sudo git clone git@github.com:codiewio/web.git .

sudo docker run --rm \
  --name go_plugin \
  --workdir /app \
  --volume $(pwd):/app \
  --volume $(pwd)/var/artefacts/plugins:/artifacts \
  golang:1.23 \
  sh -c "GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /artifacts . && sleep 10"

ls -la ./var/artefacts/plugins


# --- Playground service ---
docker run --name play_dev \
  -v ./var/artefacts/plugins/web:/hook_handler \
  -p 80:80/tcp \
  --restart on-failure \
  --network sandnet \
  -it \
  codiew/codenire-playground:latest \
  /app/codenire \
  --backend-url http://sandbox_dev/run \
  --port 80 \
  --hooks-plugins /hook_handler