output "sandbox_loadbalancer_ip" {
  value = data.digitalocean_loadbalancer.sandbox_loadbalancer.ip
}

