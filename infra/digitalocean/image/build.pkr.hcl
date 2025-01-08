source "digitalocean" "playground_droplet" {
  api_token     = var.do_token
  region        = var.do_region
  image         = "ubuntu-24-04-x64	"
  size          = "s-1vcpu-1gb"
  ssh_username  = "root"
  snapshot_name = "codenire_playground_image"
  droplet_name  = "codenire-playground-droplet"
}

source "digitalocean" "sandbox_droplet" {
  api_token     = var.do_token
  region        = var.do_region
  image         = "ubuntu-20-04-x64"
  size          = "s-1vcpu-1gb"
  ssh_username  = "root"
  snapshot_name = "codenire_sandbox_image"
  droplet_name  = "codenire-sandbox-droplet"
}

build {
  sources = [
    "source.digitalocean.playground_droplet",
    "source.digitalocean.sandbox_droplet"
  ]

  provisioner "shell" {
    inline = [
      "sudo mkdir /ops",
      "sudo chmod 777 /ops",
    ]
  }

  provisioner "file" {
    source      = "../shared/"
    destination = "/ops"
  }

  provisioner "shell" {
    script = "../shared/scripts/setup.sh"
  }

  provisioner "shell" {
    script = "../shared/scripts/${source.name}.sh"
  }
}

packer {
  required_plugins {
    digitalocean = {
      version = ">= 1.0.0"
      source  = "github.com/digitalocean/digitalocean"
    }
  }
}