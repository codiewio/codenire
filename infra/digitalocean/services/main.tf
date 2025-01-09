terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }

  cloud {
    organization = "codenire"

    workspaces {
      name = "codenire-services"
    }
  }
}

provider "digitalocean" {
  token = var.do_token
}

