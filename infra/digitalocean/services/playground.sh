#!/bin/bash

set -e

# Main Playground
docker pull codiew/codenire-playground:latest

docker ps -a --filter "name=play_dev" -q | xargs docker stop || true
docker ps -a --filter "name=play_dev" -q | xargs docker rm || true
docker ps -a --filter "name=traefik" -q | xargs docker stop || true
docker ps -a --filter "name=traefik" -q | xargs docker rm || true

docker network create play-network || true

# 80 should be open also (for letsencrypt challenge)
# TODO:: сделать условие запуска траефика по условию домена
docker run -d \
  --name traefik \
  --network play-network \
  -p 80:80 \
  -p 443:443 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /letsencrypt:/letsencrypt \
  \
  traefik:v2.11 \
  \
  --entryPoints.web.address=:80 \
  --entryPoints.websecure.address=:443 \
  --entryPoints.api.address=:8085 \
  --api.dashboard=false \
  --api.insecure=false \
  --providers.docker=true \
  --providers.docker.exposedByDefault=false \
  --certificatesResolvers.myresolver.acme.email="${2}" \
  --certificatesResolvers.myresolver.acme.storage=/letsencrypt/acme.json \
  --certificatesResolvers.myresolver.acme.httpChallenge.entryPoint=web

docker run -d --name play_dev \
  --network play-network \
  --add-host=sandbox-host:"$1" \
  --entrypoint "/playground" \
  --restart always \
  --label "traefik.enable=true" \
  --label "traefik.http.routers.play_dev.tls.certresolver=myresolver" \
  --label "traefik.http.routers.play_dev.rule=Host(\`codenire.com\`)" \
  --label "traefik.http.routers.play_dev.entrypoints=websecure" \
  --label "traefik.http.services.play_dev.loadbalancer.server.port=80" \
  \
  codiew/codenire-playground:latest \
  \
  --backend-url "http://sandbox-host:80/run" \
  --port 80

# Show start logs
sleep 3
docker ps -q --filter "name=play_dev" | xargs -I {}  docker logs --since 20s {}

