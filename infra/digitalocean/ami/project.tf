resource "digitalocean_project" "codenire_project" {
  name        = "Codenire ${local.project_env}"
  description = "This is Codenire Project"
  environment = local.project_env
}

resource "digitalocean_project_resources" "project_binding" {
  count   = local.domain_exists ? 1 : 0
  project = digitalocean_project.codenire_project.id

  resources = concat(
    digitalocean_droplet.sandbox_servers.*.urn,
    [digitalocean_droplet.playground_server.urn],
    [digitalocean_loadbalancer.sandbox_internal_loadbalancer.urn],
      local.domain_exists ? ["do:domain:${var.playground_domain}"] : [],
  )
}
