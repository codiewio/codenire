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

resource "null_resource" "run_playground" {
  # Trigger every terraform apply
  triggers = {
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
      "/tmp/script.sh",
    ]
  }
}


