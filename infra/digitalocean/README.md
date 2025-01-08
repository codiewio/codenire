# Codenire Infra

## Deployment Prerequisites
- Docker
- DigitalOcean token
- 30 minutes

## Setup environment
There are quite a few tools used for deploying this architecture so it is therefore recommended to use docker for a consistent deployment environment.

```bash
# Build the docker image
docker build -t codenire-deploy .

# Run the docker image and mount this repo into it. The ports are so that
# we can access the UI for Nomad, Vault, Consul, Traefik etc
docker run \
	-e DO_TOKEN="REPLACE_ME_WITH_DIGITAL_OCEAN_TOKEN"  \
	-v $(pwd):/codenire-deploy  \
	-it codenire-deploy
	
# Move into deploy directory
cd /codenire-deploy
```

## Build the Droplet Image
Packer is the go-to tool for creating immutable machine images. We will use it to create
the image which our cluster droplets consists of.
```
cd image && \
    packer init . && \
    packer build . && \
    cd ..
```


## Cluster infrastructure
We will use terraform to deploy the droplets, configure the firewall and vpc of the cluster.

```bash
cd ami

# Init terraform
terraform init

# Deploy droplets
terraform apply
```
