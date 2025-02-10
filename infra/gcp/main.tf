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
      name = "gcp"
    }
  }
}

provider "google" {
  project = var.project
  region  = var.region
  credentials = var.gcp_credentials
}
