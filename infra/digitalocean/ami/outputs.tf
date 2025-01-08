
output "codenire_site" {
  value = digitalocean_floating_ip.codenire_ip.ip_address
}

output "sandbox_droplet_ips" {
  value = join(",", digitalocean_droplet.sandbox_server[*].ipv4_address_private)
}

output "playground_droplet_ip" {
  value = digitalocean_droplet.playground_server.ipv4_address_private
}
