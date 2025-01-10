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
      name = "service"
    }
  }
}

provider "digitalocean" {
  token = var.do_token
}

data "tfe_outputs" "codenire_workspace_data" {
  organization = "codenire"
  workspace = "droplets"
}

locals {
  do_ssh_private = data.tfe_outputs.codenire_workspace_data.values.do_ssh_private
}