terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.47"
    }
  }

  # Work only with organization "codenire" and worspaces: ["droplets", "services"]
  cloud {
    organization = "codenire"

    workspaces {
      name = "service"
    }
  }
}

provider "digitalocean" {
  token = var.do_token
}
