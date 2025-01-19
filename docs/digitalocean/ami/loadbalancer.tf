resource "digitalocean_loadbalancer" "sandbox_internal_loadbalancer" {
  name   = "sandbox-loadbalancer-${var.environment}"
  region = var.do_region
  vpc_uuid = digitalocean_vpc.codenire_vpc.id
  disable_lets_encrypt_dns_records = true
  size_unit = 1

  network = "INTERNAL"

  droplet_ids = local.sandbox_droplet_ids

  forwarding_rule {
    entry_port     = 80
    entry_protocol = "http"

    target_port     = 80
    target_protocol = "http"
  }

  healthcheck {
    port     = 22
    protocol = "tcp"
  }
}