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



## Set Up Infrastructure
We will use terraform to deploy the droplets, configure the firewall and vpc of the cluster.

```bash
cd ami

# Init terraform (flag -cloud=false is required, see main.tf comment)
terraform init -cloud=false

# Deploy droplets
terraform apply

cd ..
```

**Awesome!** Your infra ready to install services, you can see infra details about your VM, private networking and load_balancers in terminal output.



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

**Awesome!** Your project is ready, you can see available IP address in terminal output.


## Adding HTTPS

If you registered domain and linked it to your DO account (added in DO panel and configured NS records) you can link it in deploy.

- Set in your .env file:
  - `PLAYGROUND_DOMAIN=your_domain.com` 
  - `LETSENCRYPT_EMAIL=your_email@gmail.com`
- Then rebuild docker (see "Setup environment")
- Restart deploy (see "Set Up Infrastructure" and Services Deployment")

**Awesome!** You have ready for production web app with Codenire Playground.
