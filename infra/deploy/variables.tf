variable "gcp_credentials" {
  type        = string
  sensitive   = true
  description = "Google Cloud service account credentials"
}

variable "project" {
  type    = string
  default = "codenire"
}

variable "region" {
  default = "us-east1"
}


variable "ssh_user" {
  default = "codenire"
  type    = string
}

variable "ssh_private_key" {
  default = "id_ed25519"
  type    = string
}
# ---------------------------------------------


variable "playground_domain" {
  type    = string
  default = "site.com"
}
variable "letsencrypt_email" {
  type    = string
  default = "mail@mail.com"
}

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

variable "app_version" {
  type = string
  default = "latest"
}

variable "s3_dockerfiles_endpoint" {
  type = string
  default = ""
}
variable "s3_dockerfiles_bucket" {
  type = string
  default = ""
}
variable "s3_dockerfiles_prefix" {
  type = string
  default = ""
}
variable "allow_hosts" {
  type = string
  default = ""
}

