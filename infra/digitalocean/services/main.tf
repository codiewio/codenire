terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.47"
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

data "tfe_outputs" "codenire_workspace_data" {
  organization = "codenire"
  workspace = "codenire-workspace"
}

locals {
  private_key_pem = data.tfe_outputs.codenire_workspace_data.values.private_key_pem
}