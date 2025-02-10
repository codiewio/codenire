#!/bin/bash

set -e

# ---------------------------------------------------------------------------------------------

docker system prune -a --volumes -f
echo "Docker prune finished"
echo "App version: ${tf_app_version}"

docker pull codiew/codenire-sandbox:${tf_app_version}
docker pull codiew/codenire-playground:${tf_app_version}
docker pull codiew/codenire-deproxy:${tf_app_version}
docker pull halabooda/halabooda-sandbox:latest

# Stop and Remove old container
docker ps -a \
    --filter "name=sandbox_dev" \
    --filter "name=isolated_gateway" \
    --filter "name=play_dev" \
    --filter "name=traefik" \
    -q | xargs docker rm -f || true


docker network create -d bridge isolated_net || true
docker network create play_network || true

# ---------------------------------------------------------------------------------------------

# deproxy start
docker run -d \
  --name isolated_gateway \
  --restart always \
  --network isolated_net \
  -e ALLOW_HOSTS="${tf_allow_hosts}" \
  -p 3128:3128 \
  codiew/codenire-deproxy:${tf_app_version}

# ---------------------------------------------------------------------------------------------

# Start app
docker run -d \
  --network isolated_net \
  --network play_network \
  --name sandbox_dev \
  -e HTTP_PROXY="http://isolated_gateway:3128" \
  -e HTTPS_PROXY="http://isolated_gateway:3128" \
  -e AWS_ACCESS_KEY_ID="${tf_aws_access_key_id}" \
  -e AWS_SECRET_ACCESS_KEY="${tf_aws_secret_access_key}" \
  -e AWS_REGION="${tf_aws_region}" \
  \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --restart=always \
  --restart=always \
  --privileged \
  --entrypoint "/usr/local/bin/sandbox" \
  \
  codiew/codenire-sandbox:${tf_app_version} \
  \
  --isolatedNetwork isolated_net \
  --isolatedGateway http://isolated_gateway:3128 \
  --s3DockerfilesEndpoint "${tf_s3_dockerfiles_endpoint}" \
  --s3DockerfilesBucket "${tf_s3_dockerfiles_bucket}" \
  --s3DockerfilesPrefix "${tf_s3_dockerfiles_prefix}" \
  --replicaContainerCnt 3 \
  --port 8081 \
  --isolated

echo "Sandbox started!"
# ---------------------------------------------------------------------------------------------

sudo mkdir -p /home/${tf_ssh_user}/letsencrypt
sudo touch /home/${tf_ssh_user}/letsencrypt/acme.json
sudo chmod 600 /home/${tf_ssh_user}/letsencrypt/acme.json

docker run -d \
  --name traefik \
  --network play_network \
  -p 80:80 \
  -p 443:443 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /home/${tf_ssh_user}/letsencrypt:/letsencrypt \
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
  --certificatesResolvers.myresolver.acme.email="${tf_letsencrypt_email}" \
  --certificatesResolvers.myresolver.acme.storage=/letsencrypt/acme.json \
  --certificatesResolvers.myresolver.acme.httpChallenge.entryPoint=web

echo "Traefik started!"

# ---------------------------------------------------------------------------------------------

# Build Plugin
PLUGIN_DIR=/home/${tf_ssh_user}/ops/plugin

sudo rm -rf "$PLUGIN_DIR" || true
mkdir -p "$PLUGIN_DIR" || true
docker rm -f temp_plugin_container || true
docker run -d --name temp_plugin_container halabooda/halabooda-sandbox:latest tail -f /dev/null
docker cp temp_plugin_container:/plugin/plugin "$PLUGIN_DIR"/plugin
docker rm -f temp_plugin_container || true

sudo mount -o remount,exec /home
sudo chown -R root:root "$PLUGIN_DIR"
sudo chmod +x "$PLUGIN_DIR"/plugin

echo "Plugin ready on $PLUGIN_DIR"
ls -la "$PLUGIN_DIR"

# ---------------------------------------------------------------------------------------------

docker run -d --name play_dev \
  --network play_network \
  --network isolated_net \
  --entrypoint "/playground" \
  --restart always \
  --label "traefik.enable=true" \
  --label "traefik.http.routers.play_dev.tls.certresolver=myresolver" \
  --label "traefik.http.routers.play_dev.rule=Host(\"${tf_playground_domain}\")" \
  --label "traefik.http.routers.play_dev.entrypoints=websecure" \
  --label "traefik.http.services.play_dev.loadbalancer.server.port=80" \
  \
  -v "$PLUGIN_DIR"/plugin:/plugin \
  \
  codiew/codenire-playground:${tf_app_version} \
  \
  --backend-url "http://sandbox_dev:8081" \
  --port 80 \
  --external-templates go,go-prev \
  --throttle-limit 50

echo "Playground started!"
echo "Domain: ${tf_playground_domain}"

# ---------------------------------------------------------------------------------------------

# Show start logs
sleep 10
docker ps -q --filter "name=sandbox_dev" | xargs -I {}  docker logs --since 20s {}
docker ps -q --filter "name=play_dev" | xargs -I {}  docker logs --since 20s {}
