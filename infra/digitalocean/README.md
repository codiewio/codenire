# Codenire Infra

## Deployment Prerequisites
- 30 minutes
- Docker
- DigitalOcean token
- Terraform Profile (Free)
  - Need create Organization and 2 workspaces — need for settings and control deploy
  - Need generate User API token [link](https://app.terraform.io/app/settings/tokens) — need for manage infra/services state

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

# Init terraform
terraform init

# Deploy droplets
terraform apply

cd ..
```


## Services Deployment
We will use terraform to deploy the services and link playground with sandbox backend via private sandbox IP

```bash
cd services

# Init terraform
terraform init

# Deploy services
terraform apply

cd ..
```

[!] Sandbox use https://github.com/codiewio/dockerfiles for default source of containers which stared in sandbox. 
If you would like replace it with your source just call `terraform apply` command with you source. 
Example: `terraform apply -var="dockerfiles_git_repo=https://github.com/USERNAME/REPONAME"` (repos should be public and use HTTPS strongly)
