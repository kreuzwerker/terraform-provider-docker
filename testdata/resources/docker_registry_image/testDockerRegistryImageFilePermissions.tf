provider "docker" {
  alias = "private"
  registry_auth {
    address = "%s"
  }
}

resource "docker_registry_image" "file_permissions" {
  provider             = "docker.private"
  name                 = "%s"
  insecure_skip_verify = true

  build {
    context      = "%s"
    dockerfile   = "%s"
  }
}
