terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.47"
    }

    hcp = {
      source = "hashicorp/hcp"
      version = "~> 0.8"
    }
  }

  cloud {
    organization = "codenire"

    workspaces {
      name = "droplets"
    }
  }
}

provider "digitalocean" {
  token = var.do_token
}

locals {
  input_environment_enums = {
    dev = "Development",
    prod = "Production",
    stage = "Staging"
  }
  project_env = local.input_environment_enums[var.environment]
}

data "digitalocean_images" "playground_images" {
  filter {
    key    = "private"
    values = ["true"]
  }
  filter {
    key    = "name"
    values = ["codenire_playground_image"]
  }
  sort {
    key       = "created"
    direction = "desc"
  }
}

data "digitalocean_images" "sandbox_images" {
  filter {
    key    = "private"
    values = ["true"]
  }
  filter {
    key    = "name"
    values = ["codenire_sandbox_image"]
  }
  sort {
    key       = "created"
    direction = "desc"
  }
}

resource "digitalocean_droplet" "sandbox_servers" {
  count = var.sandbox_servers_count
  image = data.digitalocean_images.sandbox_images.images[0].id
  name   = "sandbox-server-${var.environment}-${count.index}"
  region = var.do_region
  size   = var.sandbox_droplet_size
  ssh_keys  = [digitalocean_ssh_key.codenire_ssh.fingerprint]
  vpc_uuid  = digitalocean_vpc.codenire_vpc.id
  ipv6     = false
  # monitoring = true

  tags = [
    local.retry_join.tag_name,
    "${local.retry_join.tag_name}_${var.environment}",
    "${local.retry_join.tag_name}_sandbox"
  ]
}

resource "digitalocean_droplet" "playground_server" {
  image = data.digitalocean_images.playground_images.images[0].id
  name     = "playground-server-${var.environment}"
  region   = var.do_region
  size   = var.playground_droplet_size
  ssh_keys  = [digitalocean_ssh_key.codenire_ssh.fingerprint]
  vpc_uuid = digitalocean_vpc.codenire_vpc.id
  # monitoring = true

  tags = [
    local.retry_join.tag_name,
    "${local.retry_join.tag_name}_${var.environment}",
    "${local.retry_join.tag_name}_playground"
  ]
}

# resource "digitalocean_project" "codenire_project" {
#   name        = "Codenire ${local.project_env}"
#   description = "This is Codenire Project"
#   environment = local.project_env
#
#   resources   = concat(
#     digitalocean_droplet.sandbox_servers.*.urn,
#     [digitalocean_droplet.playground_server.urn],
#   )
# }

locals {
  sandbox_droplet_ids = digitalocean_droplet.sandbox_servers.*.id

  all_droplets = concat(
    local.sandbox_droplet_ids,
    [digitalocean_droplet.playground_server.id]
  )

  all_ips = ["0.0.0.0/0", "::/0"]

  ssh_addresses = var.environment == "dev" ? local.all_ips : local.all_droplets
}

resource "digitalocean_loadbalancer" "sandbox_internal_loadbalancer" {
  name   = "sandbox-loadbalancer-${var.environment}"
  region = var.do_region
  # project_id = digitalocean_project.codenire_project.id
  vpc_uuid = digitalocean_vpc.codenire_vpc.id
  disable_lets_encrypt_dns_records = true
  size_unit = 1

  network = "INTERNAL"

  droplet_ids = local.sandbox_droplet_ids

  forwarding_rule {
    entry_port     = 80
    entry_protocol = "http"

    target_port     = 80
    target_protocol = "http"
  }

  healthcheck {
    port     = 22
    protocol = "tcp"
  }
}
