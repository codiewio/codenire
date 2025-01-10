variable "do_token" {
  type = string
}

variable do_ssh_private_key {
  type = string
}

variable "environment" {
  type    = string
  description = "input environment, allowed values are dev, stage or prod"
  default = "dev"
}

variable "dockerfiles_repository" {
  default = "https://github.com/codiewio/dockerfiles.git"
}

variable "do_region" {
  type    = string
  default = "nyc1"
}

locals {
  retry_join = {
    provider  = "digitalocean"
    region    = var.do_region
    tag_name  = "codenire"
    api_token = var.do_token
  }
}