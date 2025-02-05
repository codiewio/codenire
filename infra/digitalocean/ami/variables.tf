variable "do_token" {
  type = string
}

# variable do_ssh_public_key {
#   type = string
# }

variable "environment" {
  type    = string
  description = "input environment, allowed values are dev, stage or prod"
  default = "dev"
}

variable "do_region" {
  type    = string
  default = "nyc1"
}

variable "sandbox_servers_count" {
  default = 1
}

variable "sandbox_droplet_size" {
  type = string
  default = "s-1vcpu-1gb"
}
variable "playground_droplet_size" {
  type = string
  default = "s-1vcpu-1gb"
}

variable "playground_domain" {
  default = null
  nullable = true
}

locals {
  retry_join = {
    provider  = "digitalocean"
    region    = var.do_region
    tag_name  = "codenire"
    api_token = var.do_token
  }
}