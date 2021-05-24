provider "docker" {
  alias = "private"
  registry_auth {
    address = "%s"
  }
}
resource "docker_registry_image" "foo" {
  provider             = "docker.private"
  name                 = "%s"
  insecure_skip_verify = true

  build {
    context      = "%s"
    remove       = true
    force_remove = true
    no_cache     = true
  }
}
