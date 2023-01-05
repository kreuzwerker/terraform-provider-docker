provider "docker" {
  alias = "private"
}
resource "docker_image" "foo" {
  provider             = "docker.private"
  name                 = "%s"

  build {
    context      = "%s"
    remove       = true
    force_remove = true
    no_cache     = true
  }
}
