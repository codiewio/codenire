data "digitalocean_droplets" "playground_droplets" {
  filter {
    key    = "tags"
    values = ["${local.retry_join.tag_name}_${var.environment}"]
  }
  filter {
    key = "tags"
    values = ["${local.retry_join.tag_name}_playground"]
  }
}

data "digitalocean_loadbalancer" "sandbox_loadbalancer" {
  # name from /ami/main.tf -> digitalocean_loadbalancer.sandbox_internal_loadbalancer
  name = "sandbox-loadbalancer-${var.environment}"
}

locals {
  sandbox_balancer_ip = data.digitalocean_loadbalancer.sandbox_loadbalancer.ip
}

resource "null_resource" "run_playground" {
  # Trigger every terraform apply
  triggers = {
    always_run = timestamp()
  }

  count = length(data.digitalocean_droplets.playground_droplets.droplets)

  connection {
    type        = "ssh"
    user        = "root"
    private_key = local.do_ssh_private
    host        = data.digitalocean_droplets.playground_droplets.droplets[count.index].ipv4_address
  }

  provisioner "file" {
    source      = "playground.sh"
    destination = "/tmp/script.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/script.sh",
      "/tmp/script.sh ${local.sandbox_balancer_ip}",
    ]
  }
}
