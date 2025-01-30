data "digitalocean_droplets" "sandbox_droplets" {
  filter {
    key    = "tags"
    values = ["${local.retry_join.tag_name}_${var.environment}"]
  }
  filter {
    key = "tags"
    values = ["${local.retry_join.tag_name}_sandbox"]
  }
}

resource "null_resource" "run_sandbox" {
  # Trigger every terraform apply
  triggers = {
    always_run = timestamp()
  }

  count = length(data.digitalocean_droplets.sandbox_droplets.droplets)

  connection {
    type        = "ssh"
    user        = "root"
    private_key = local.do_ssh_private
    host        = data.digitalocean_droplets.sandbox_droplets.droplets[count.index].ipv4_address
  }

  provisioner "file" {
    source      = "sandbox.sh"
    destination = "/tmp/script.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/script.sh",
      "/tmp/script.sh",
    ]
  }
}


