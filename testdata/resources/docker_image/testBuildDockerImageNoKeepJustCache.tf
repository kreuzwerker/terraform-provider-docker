provider "docker" {
  alias = "private"
}
resource "docker_image" "%s" {
  provider             = "docker.private"
  name                 = "%s"

  build {
    context      = "%s"
    remove       = false
    force_remove = false
    no_cache     = false
  }
}
