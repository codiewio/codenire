

# Firewall for private sandboxes
resource "digitalocean_firewall" "codenire_intra_traffic" {
  name = "codenire-intra-traffic"

  droplet_ids = local.sandbox_droplet_ids

  inbound_rule {
    protocol           = "tcp"
    port_range         = "1-65535"
    source_droplet_ids = local.all_droplets
  }
  inbound_rule {
    protocol           = "udp"
    port_range         = "1-65535"
    source_droplet_ids = local.all_droplets
  }
  inbound_rule {
    protocol           = "icmp"
    port_range         = "1-65535"
    source_droplet_ids = local.all_droplets
  }

  # --- [!] IMPORTANT [!] --------------
  # --- ssh access for dev env ---------
  inbound_rule {
    protocol         = "tcp"
    port_range       = "22"
    source_addresses = local.ssh_addresses
  }
  # ------------------------------------

  inbound_rule {
    protocol         = "tcp"
    port_range       = "80"
    source_droplet_ids = local.all_droplets
  }
  inbound_rule {
    protocol         = "tcp"
    port_range       = "443"
    source_droplet_ids = local.all_droplets
  }

  outbound_rule {
    protocol              = "tcp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }
  outbound_rule {
    protocol              = "udp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }
  outbound_rule {
    protocol              = "icmp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }
}

# Firewall for public playground
resource "digitalocean_firewall" "codenire_play" {
  name = "codenire-play"

  droplet_ids = [digitalocean_droplet.playground_server.id]

  # All tcp traffic on port 22, 80 and 443 from outside
  inbound_rule {
    protocol         = "tcp"
    port_range       = "22"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  # TODO:: удалить
  inbound_rule {
    protocol         = "tcp"
    port_range       = "8000-8100"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  inbound_rule {
    protocol         = "tcp"
    port_range       = "80"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }
  inbound_rule {
    protocol         = "tcp"
    port_range       = "443"
    source_addresses = ["0.0.0.0/0", "::/0"]
  }

  # All traffic from cluster
  inbound_rule {
    protocol           = "tcp"
    port_range         = "1-65535"
    source_droplet_ids = local.all_droplets
  }
  inbound_rule {
    protocol           = "udp"
    port_range         = "1-65535"
    source_droplet_ids = local.all_droplets
  }
  inbound_rule {
    protocol           = "icmp"
    port_range         = "1-65535"
    source_droplet_ids = local.all_droplets
  }

  outbound_rule {
    protocol              = "tcp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }
  outbound_rule {
    protocol              = "udp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }
  outbound_rule {
    protocol              = "icmp"
    port_range            = "1-65535"
    destination_addresses = ["0.0.0.0/0", "::/0"]
  }
}