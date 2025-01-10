# Codenire Infra

## Deployment Prerequisites
- 30 minutes
- Docker
- DigitalOcean token

## Setup environment
There are quite a few tools used for deploying this architecture so it is therefore recommended to use docker for a consistent deployment environment.

```bash
# Build the docker image
docker build -t codenire-deploy .

# Prepare you vars
cp .env.example .env 

# Run the docker image and inside you can manage of your infrastructure
docker run --env-file .env -v $(pwd):/codenire-deploy -it codenire-deploy
	
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

# Init terraform (flag -cloud=false is required, see. main.tf comment)
terraform init -cloud=false

# Deploy droplets
terraform apply

cd ..
```


## Services Deployment
We will use terraform to deploy the services and link playground with sandbox backend via private sandbox IP

```bash
cd services

# Init terraform (flag -cloud=false is required, see. main.tf comment)
terraform init -cloud=false

# Deploy services
terraform apply

cd ..
```

[!] Sandbox use https://github.com/codiewio/dockerfiles for default source of containers which stared in sandbox. 
If you would like replace it with your source you can:
1. Call `terraform apply` command with you source. 
  `terraform apply \
   -var="dockerfiles_repository=https://github.com/USERNAME/REPONAME"`
  (repos should be public and use HTTPS strongly)
2. Set TF_VAR_dockerfiles_repository in .env file and re-run Docker service
