output "ssh_private_key" {
  value = tls_private_key.ed25519_key.private_key_openssh
  sensitive = true
}

output "vm_id" {
  value = local.sandbox_ip
}