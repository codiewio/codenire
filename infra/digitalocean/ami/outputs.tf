
output "codenire_site" {
  value = digitalocean_droplet.playground_server.ipv4_address
}

output "sandbox_droplet_ips" {
  value = join(",", digitalocean_droplet.sandbox_servers[*].ipv4_address_private)
}

output "playground_droplet_ip" {
  value = digitalocean_droplet.playground_server.ipv4_address_private
}

output "sandbox_loadbalancer_ip" {
  value = digitalocean_loadbalancer.sandbox_internal_loadbalancer.ip
}
