resource "tls_private_key" "ed25519_key" {
  algorithm = "ED25519"
}

resource "google_compute_project_metadata_item" "ssh_key" {
  key   = "codenire-ssh"
  value = "Codenire SSH Key — ${var.environment}:${tls_private_key.ed25519_key.public_key_openssh}"
  project = var.project
}
