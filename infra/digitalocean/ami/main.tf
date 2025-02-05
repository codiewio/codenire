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

locals {
  sandbox_droplet_ids = digitalocean_droplet.sandbox_servers.*.id

  all_droplets = concat(
    local.sandbox_droplet_ids,
    [digitalocean_droplet.playground_server.id]
  )

  all_vpc_ipv4_private = [digitalocean_vpc.codenire_vpc.ip_range]
}

