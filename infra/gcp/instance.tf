locals {
  s3DockerfilesEndpoint = "https://nyc3.digitaloceanspaces.com"
  s3DockerfilesBucket   = "codiew"
  s3DockerfilesPrefix   = "codenire_templates"
}

resource "google_compute_instance" "sandbox_vm" {
  name = var.sandbox_name
  machine_type = var.machine_type
  zone         = var.zone

  network_interface {
    network = var.network

    access_config {
      # IP будет автоматически назначен при подключении access_config
    }
  }

  boot_disk {
    initialize_params {
      image = "cos-cloud/cos-dev"
    }
  }

  metadata_startup_script = templatefile("deploy-app.sh", {
    aws_access_key_id     = var.aws_access_key_id
    aws_secret_access_key = var.aws_secret_access_key
    aws_region            = var.aws_region

    playground_domain = var.playground_domain
    letsencrypt_email = var.letsencrypt_email

    allow_hosts = ".digitaloceanspaces.com"

    s3DockerfilesEndpoint = local.s3DockerfilesEndpoint
    s3DockerfilesBucket   = local.s3DockerfilesBucket
    s3DockerfilesPrefix   = local.s3DockerfilesPrefix

    user = var.ssh_user
  })

  metadata = {
    "user-data" = file("cloud-init.yaml")
    "ssh-keys" = "${var.ssh_user}:${tls_private_key.ed25519_key.public_key_openssh} ${local.local_public_ssh_account}"
  }

  tags = ["codenire-tag"]
}