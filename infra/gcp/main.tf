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

resource "tls_private_key" "ed25519_key" {
  algorithm = "ED25519"
}

resource "google_compute_project_metadata_item" "ssh_key" {
  key   = "codenire-ssh"
  value = "Codenire SSH Key — ${var.environment}:${tls_private_key.ed25519_key.public_key_openssh}"
  project = var.project
}

locals {
  local_user_admin = "admin"
  local_public_ssh_account = var.local_public_ssh != "" ? "${local.local_user_admin}:${var.local_public_ssh}" : ""
  sandbox_ip = google_compute_instance.sandbox_vm.network_interface[0].access_config[0].nat_ip
}

# ---------------------------------------------------------------

output "ssh_private_key" {
  value = tls_private_key.ed25519_key.private_key_pem
  sensitive = true
}

output "ssh_host" {
  value = "ssh ${local.local_user_admin}@${local.sandbox_ip}"
}

output "vm_id" {
  value = local.sandbox_ip
}