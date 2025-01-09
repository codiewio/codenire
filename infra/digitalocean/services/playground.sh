#!/bin/bash

set -e

# Hard remove all!
docker system prune -a --volumes -f

docker run -d --name play_dev \
  -p 80:80 \
  --entrypoint "/app/codenire" \
  --restart always \
  codiew/codenire-playground:latest \
  --backend-url http://sandbox_dev/run \
  --port 80