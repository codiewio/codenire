resource "google_compute_firewall" "codenire_firewall" {
  name    = "allow-ssh"
  network = var.network

  allow {
    protocol = "tcp"
    ports = ["22"]
  }

  allow {
    protocol = "tcp"
    ports = ["80"]
  }

  allow {
    protocol = "tcp"
    ports = ["443"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags = google_compute_instance.test_vm.tags
}