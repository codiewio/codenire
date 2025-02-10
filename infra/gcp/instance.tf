resource "google_compute_address" "codenire_ip" {
  name         = "codenire-ip"
  address_type = "EXTERNAL"
  region       = var.region
}

resource "google_compute_instance" "test_vm" {
  name = var.test_vm
  machine_type = var.machine_type
  zone         = var.zone

  network_interface {
    network = var.network

    access_config {
      nat_ip = google_compute_address.codenire_ip.address
    }
  }

  boot_disk {
    initialize_params {
      image = "cos-cloud/cos-dev"
    }
  }

  metadata = {
    "user-data" = file("cloud-init.yaml")
    "ssh-keys" = "${var.ssh_user}:${tls_private_key.ed25519_key.public_key_openssh}"
  }

  tags = [
    "codenire-tag",
  ]
}

locals {
  sandbox_ip = google_compute_instance.test_vm.network_interface[0].access_config[0].nat_ip
}