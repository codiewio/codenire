#!/bin/bash

set -e

# ---------------------------------------------------------------------------------------------

docker system prune -a --volumes -f
echo "Docker prune finished"

docker pull codiew/codenire-sandbox:latest
docker pull codiew/codenire-playground:latest
docker pull codiew/codenire-deproxy:latest
docker pull halabooda/halabooda-sandbox:latest

docker network create app_network || true
docker network create --driver bridge --subnet=192.168.100.0/24 isolated_net || true
docker network create play-network || true

# Stop and Remove old container
docker ps -a \
    --filter "name=sandbox_dev" \
    --filter "name=isolated_gateway" \
    --filter "name=play_dev" \
    --filter "name=traefik" \
    -q | xargs docker rm -f || true

# ---------------------------------------------------------------------------------------------

# Build Plugin
PLUGIN_DIR=/home/${user}/ops/plugin

rm $PLUGIN_DIR || true
mkdir -p $PLUGIN_DIR || true
docker rm -f temp_plugin_container || true
docker run -d --name temp_plugin_container halabooda/halabooda-sandbox:latest tail -f /dev/null
docker cp temp_plugin_container:/plugin/plugin $PLUGIN_DIR/plugin
docker rm -f temp_plugin_container || true

echo "Plugin ready on $PLUGIN_DIR"
ls -la $PLUGIN_DIR

# ---------------------------------------------------------------------------------------------

# deproxy start
docker run -d \
  --name isolated_gateway \
  --restart always \
  --network isolated_net \
  -e ALLOW_HOSTS=${allow_hosts} \
  -p 3128:3128 \
  codiew/codenire-deproxy:latest

# ---------------------------------------------------------------------------------------------

# Start app
docker run -d \
  -p 8081:8081 \
  --network isolated_net \
  --network play-network \
  --name sandbox_dev \
  -e HTTP_PROXY="http://isolated_gateway:3128" \
  -e HTTPS_PROXY="http://isolated_gateway:3128" \
  -e AWS_ACCESS_KEY_ID=${aws_access_key_id} \
  -e AWS_SECRET_ACCESS_KEY=${aws_secret_access_key} \
  -e AWS_REGION=${aws_region} \
  \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --restart=always \
  --restart=always \
  --privileged \
  --entrypoint "/usr/local/bin/sandbox" \
  \
  codiew/codenire-sandbox:latest \
  \
  --isolatedNetwork isolated_net \
  --isolatedGateway http://isolated_gateway:3128 \
  --s3DockerfilesEndpoint ${s3DockerfilesEndpoint} \
  --s3DockerfilesBucket ${s3DockerfilesBucket} \
  --s3DockerfilesPrefix ${s3DockerfilesPrefix} \
  --replicaContainerCnt 1 \
  --port 8081 \
  --isolated

echo "Sandbox started!"
# ---------------------------------------------------------------------------------------------

#mkdir -p /home/${user}/letsencrypt
#touch /home/${user}/letsencrypt/acme.json
#chmod 600 /home/${user}/letsencrypt/acme.json
#
## 80 should be open also (for letsencrypt challenge)
#docker run -d \
#  --name traefik \
#  --network play-network \
#  -p 80:80 \
#  -p 443:443 \
#  -v /var/run/docker.sock:/var/run/docker.sock \
#  -v /home/${user}/letsencrypt:/letsencrypt \
#  \
#  traefik:v2.11 \
#  \
#  --entryPoints.web.address=:80 \
#  --entryPoints.websecure.address=:443 \
#  --entryPoints.api.address=:8085 \
#  --api.dashboard=false \
#  --api.insecure=false \
#  --providers.docker=true \
#  --providers.docker.exposedByDefault=false \
#  --certificatesResolvers.myresolver.acme.email="${letsencrypt_email}" \
#  --certificatesResolvers.myresolver.acme.storage=/letsencrypt/acme.json \
#  --certificatesResolvers.myresolver.acme.httpChallenge.entryPoint=web
#
#echo "Traefik started!"

# ---------------------------------------------------------------------------------------------

#  --label "traefik.http.routers.play_dev.rule=Host(\`${playground_domain}\`)" \
#  --label "traefik.http.routers.play_dev.entrypoints=websecure" \

docker run -d --name play_dev \
  -p 80:80 \
  --network play-network \
  --entrypoint "/playground" \
  --restart always \
  --label "traefik.enable=true" \
  --label "traefik.http.routers.play_dev.tls.certresolver=myresolver" \
  --label "traefik.http.routers.play_dev.rule=PathPrefix(``/``)" \
  --label "traefik.http.routers.play_dev.entrypoints=web" \
  --label "traefik.http.services.play_dev.loadbalancer.server.port=80" \
  \
  -v /home/${user}/plugin/plugin:/plugin \
  \
  codiew/codenire-playground:latest \
  \
  --backend-url "http://sandbox_dev:8081" \
  --port 80 \
  --external-templates go,go-prev

echo "Playground started!"

# ---------------------------------------------------------------------------------------------

# Show start logs
sleep 10
docker ps -q --filter "name=sandbox_dev" | xargs -I {}  docker logs --since 20s {}
docker ps -q --filter "name=play_dev" | xargs -I {}  docker logs --since 20s {}
