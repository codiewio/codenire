# resource "digitalocean_ssh_key" "codenire_ssh" {
#   name       = "Codenire SSH Key — ${var.environment}"
#   public_key = var.do_ssh_public_key
# }

resource "tls_private_key" "rsa_key" {
  algorithm = "ED25519"
}

resource "digitalocean_ssh_key" "codenire_ssh" {
  name       = "Codenire SSH Key — ${var.environment}"
  public_key = tls_private_key.rsa_key.public_key_openssh
}