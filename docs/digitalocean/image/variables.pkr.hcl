variable "do_token" {
  type = string
}

variable "do_region" {
  type = string
  default = "nyc1"
}

variable "droplet_name" {
  type = string
  default = "codenire-playground"
}

variable "snapshot_name" {
  type = string
  default = "codenire_image"
}