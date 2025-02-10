variable "gcp_credentials" {
  type        = string
  sensitive   = true
  description = "Google Cloud service account credentials"
}

variable "project" {
  type    = string
  default = "codenire"
}

variable "machine_type" {
  type    = string
  default = "e2-machine"
}

variable "region" {
  default = "us-east1"
}

variable "zone" {
  type    = string
  default = "us-east1-c"
}

variable "environment" {
  type        = string
  description = "input environment, allowed values are dev, stage or prod"
  default     = "dev"
}

variable "test_vm" {
  type    = string
  default = "codenire-dev-vm"
}

variable "network" {
  type    = string
  default = "default"
}

variable "ssh_user" {
  default = "codenire"
  type    = string
}

variable "ssh_private_key" {
  default = "id_ed25519"
  type    = string
}
