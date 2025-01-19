
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

resource "digitalocean_droplet" "sandbox_servers" {
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
    local.retry_join.tag_name,
    "${local.retry_join.tag_name}_${var.environment}",
    "${local.retry_join.tag_name}_sandbox"
  ]
}

resource "digitalocean_droplet" "playground_server" {
  image = data.digitalocean_images.playground_images.images[0].id
  name     = "playground-server-${var.environment}"
  region   = var.do_region
  size   = var.playground_droplet_size
  ssh_keys  = [digitalocean_ssh_key.codenire_ssh.fingerprint]
  vpc_uuid = digitalocean_vpc.codenire_vpc.id
  # monitoring = true

  tags = [
    local.retry_join.tag_name,
    "${local.retry_join.tag_name}_${var.environment}",
    "${local.retry_join.tag_name}_playground"
  ]
}
