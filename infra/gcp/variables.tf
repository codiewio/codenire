variable "gcp_credentials" {
  type        = string
  sensitive   = true
  description = "Google Cloud service account credentials"
}

variable "machine_type" {
  type = string
  default = "e2-medium"
}

variable "project" {
  type    = string
  default = "codenire"
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

variable "sandbox_name" {
  type    = string
  default = "codenire-sandbox-vm"
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

variable "local_public_ssh" {
  type    = string
  default = ""
}

variable "playground_domain" {
  type    = string
  default = "codenire"
}
variable "letsencrypt_email" {
  type    = string
  default = "mail@mail.com"
}

# ---------------------------------------------

variable "aws_access_key_id" {
  type    = string
  default = ""
}

variable "aws_secret_access_key" {
  type    = string
  default = ""
}

variable "aws_region" {
  type    = string
  default = ""
}

