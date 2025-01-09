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

locals {
  sandbox_ip = data.digitalocean_droplets.sandbox_droplets.droplets[0].ipv4_address_private
}

resource "null_resource" "run_playground" {
  # Trigger every terraform apply
  triggers = {
    # TODO:: tags from Github
    always_run = timestamp()
  }

  count = length(data.digitalocean_droplets.playground_droplets.droplets)

  connection {
    type        = "ssh"
    user        = "root"
    private_key = var.do_ssh_key
    host        = data.digitalocean_droplets.playground_droplets.droplets[count.index].ipv4_address
  }

  provisioner "file" {
    source      = "playground.sh"
    destination = "/tmp/script.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/script.sh",
      "/tmp/script.sh ${local.sandbox_ip}",
    ]
  }
}
