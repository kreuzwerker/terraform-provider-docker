provider "docker" {
  alias = "private"
  registry_auth {
    address = "%s"
  }
}
data "docker_registry_image" "foo_private" {
  provider             = "docker.private"
  name                 = "%s"
  insecure_skip_verify = true
}
resource "docker_image" "foo_private" {
  provider      = "docker.private"
  name          = data.docker_registry_image.foo_private.name
  keep_locally  = true
  pull_triggers = [data.docker_registry_image.foo_private.sha256_digest]
}
