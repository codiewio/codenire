terraform {
  cloud {

    organization = "codenire"

    workspaces {
      name = "codenire-services"
    }
  }
}