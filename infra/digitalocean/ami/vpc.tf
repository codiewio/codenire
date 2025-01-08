resource "digitalocean_vpc" "codenire_vpc" {
  name     = "codenire-vpc"
  region   = var.do_region
  ip_range = "10.0.0.0/24"
}