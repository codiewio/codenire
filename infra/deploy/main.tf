terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.47"
    }

    hcp = {
      source  = "hashicorp/hcp"
      version = "~> 0.8"
    }
  }

  cloud {
    organization = "codenire"
    workspaces {
      name = "gcp-service"
    }
  }
}

provider "google" {
  project     = var.project
  region      = var.region
  credentials = var.gcp_credentials
}

data "tfe_outputs" "codenire_machine_data" {
  organization = "codenire"
  workspace    = "gcp"
}

locals {
  sandbox_machine_ip   = data.tfe_outputs.codenire_machine_data.values.vm_id
  sandbox_user         = var.ssh_user
  sandbox_user_private = data.tfe_outputs.codenire_machine_data.values.ssh_private_key
}


locals {
  install-service-script = templatefile("${path.module}/start-service.sh", {
    tf_aws_access_key_id       = var.aws_access_key_id
    tf_aws_secret_access_key   = var.aws_secret_access_key
    tf_aws_region              = var.aws_region
    tf_playground_domain       = var.playground_domain
    tf_letsencrypt_email       = var.letsencrypt_email
    tf_allow_hosts             = var.allow_hosts
    tf_s3_dockerfiles_endpoint = var.s3_dockerfiles_endpoint
    tf_s3_dockerfiles_bucket   = var.s3_dockerfiles_bucket
    tf_s3_dockerfiles_prefix   = var.s3_dockerfiles_prefix
    tf_ssh_user                = var.ssh_user
    tf_app_version             = var.app_version
  })
}

output "install-service-script" {
  value = local.install-service-script
  sensitive = true
}

output "sandbox_ip" {
  value = local.sandbox_machine_ip
  sensitive = true
}

output "sandbox_user" {
  value = local.sandbox_user
  sensitive = true
}

output "ssh_user_private" {
  value = local.sandbox_user_private
  sensitive = true
}
