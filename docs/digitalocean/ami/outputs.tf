output "sandbox_droplet_ips" {
  value = join(",", digitalocean_droplet.sandbox_servers[*].ipv4_address_private)
}

output "playground_droplet_ip" {
  value = digitalocean_droplet.playground_server.ipv4_address
}

output "sandbox_loadbalancer_ip" {
  value = digitalocean_loadbalancer.sandbox_internal_loadbalancer.ip
}

output "do_ssh_public" {
  value = tls_private_key.rsa_key.public_key_openssh
}

output "do_ssh_private" {
  value     = tls_private_key.rsa_key.private_key_pem
  sensitive = true
}

output "playground_url" {
  value = local.domain_exists ? var.playground_domain : digitalocean_droplet.playground_server.ipv4_address
}
