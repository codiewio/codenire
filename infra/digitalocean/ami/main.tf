terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

# Configure the DigitalOcean Provider
provider "digitalocean" {
  token = var.do_token
}

locals {
  input_environment_enums = {
    dev = "Development",
    prod = "Production",
    stage = "Staging"
  }
  project_env = local.input_environment_enums[var.environment]
}

data "digitalocean_images" "playground_images" {
  filter {
    key    = "private"
    values = ["true"]
  }
  filter {
    key    = "name"
    values = ["codenire_playground_image"]
  }
  sort {
    key       = "created"
    direction = "desc"
  }
}

data "digitalocean_images" "sandbox_images" {
  filter {
    key    = "private"
    values = ["true"]
  }
  filter {
    key    = "name"
    values = ["codenire_sandbox_image"]
  }
  sort {
    key       = "created"
    direction = "desc"
  }
}

resource "digitalocean_ssh_key" "codenire_ssh" {
  name       = "Codenire SSH Key"
  public_key = file("${var.shared_path}/config/id_rsa.pub")
}

resource "digitalocean_droplet" "sandbox_server" {
  count = var.sandbox_servers_count
  image = data.digitalocean_images.sandbox_images.images[0].id
  name   = "sandbox-server-${var.environment}-${count.index}"
  region = var.do_region
  size   = var.sandbox_droplet_size
  ssh_keys  = [digitalocean_ssh_key.codenire_ssh.fingerprint]
  vpc_uuid  = digitalocean_vpc.codenire_vpc.id
  ipv6     = false
  # monitoring = true

  tags = [
    "${local.retry_join.tag_name}_sandbox",
    "${local.retry_join.tag_name}_${var.environment}"
  ]
}

resource "digitalocean_droplet" "playground_server" {
  # count = var.playground_servers_count
  image = data.digitalocean_images.playground_images.images[0].id
  name     = "playground-server-${var.environment}"
  region   = var.do_region
  size   = var.playground_droplet_size
  ssh_keys  = [digitalocean_ssh_key.codenire_ssh.fingerprint]
  vpc_uuid = digitalocean_vpc.codenire_vpc.id
  # monitoring = true

  tags = [
    "${local.retry_join.tag_name}_playground",
    "${local.retry_join.tag_name}_${var.environment}"

  ]
}


resource "digitalocean_project" "codenire_project" {
  name        = "Codenire ${local.project_env}"
  description = "This is Codenire Project"
  environment = local.project_env

  # TODO:: filter droplets by tag (environment)
  # https://chatgpt.com/share/677d64a4-68cc-800c-b321-540db0cefd28
  resources   = concat(
    digitalocean_droplet.sandbox_server.*.urn,
    [digitalocean_droplet.playground_server.urn]
  )
}

resource "digitalocean_floating_ip" "codenire_ip" {
  region = var.do_region
}

resource "digitalocean_floating_ip_assignment" "codenire_web" {
  ip_address = digitalocean_floating_ip.codenire_ip.ip_address
  droplet_id = digitalocean_droplet.playground_server.id
}

locals {
  sandbox_droplet_ids = concat(
    digitalocean_droplet.sandbox_server.*.id
  )

  all_droplets = concat(
    local.sandbox_droplet_ids,
    [digitalocean_droplet.playground_server.id]
  )
}

resource "digitalocean_loadbalancer" "sandbox_internal" {
  name   = "sandbox-loadbalancer"
  region = var.do_region
  project_id = digitalocean_project.codenire_project.id
  vpc_uuid = digitalocean_vpc.codenire_vpc.id

  disable_lets_encrypt_dns_records = true

  # network = "INTERNAL"

  droplet_ids = local.sandbox_droplet_ids

  forwarding_rule {
    entry_port     = 80
    entry_protocol = "http"

    target_port     = 80
    target_protocol = "http"
  }

  healthcheck {
    port     = 22
    protocol = "tcp"
  }

  firewall {
    deny = ["cidr:1.2.0.0/16", "ip:2.3.4.5"]
  }
}



# Firewall
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

resource "digitalocean_firewall" "codenire_play" {
  name = "codenire-play"

  droplet_ids = [digitalocean_droplet.playground_server.id]


  # All tcp traffic on port 22, 80 and 443 from outside
  inbound_rule {
    protocol         = "tcp"
    port_range       = "22"
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