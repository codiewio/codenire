

data "digitalocean_domains" "all_domains" {}

locals {
  domain_exists = var.playground_domain != null && length([for d in data.digitalocean_domains.all_domains.domains : d if d.name == var.playground_domain]) > 0
}
