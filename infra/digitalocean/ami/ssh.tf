resource "digitalocean_ssh_key" "codenire_ssh" {
  name       = "Codenire SSH Key — ${var.environment}"
  public_key = var.do_ssh_public_key
}