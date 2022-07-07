provider "docker" {
  alias = "private"
  registry_auth {
    address = "%s"
  }
}
resource "docker_registry_image" "%s" {
  provider             = "docker.private"
  name                 = "%s"
  insecure_skip_verify = true

  build {
    context      = "%s"
    remove       = false
    force_remove = false
    no_cache     = false
  }
}
