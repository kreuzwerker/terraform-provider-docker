provider "docker" {
  alias = "private"
}

resource "docker_image" "file_permissions" {
  provider             = "docker.private"
  name                 = "%s"

  build {
    context      = "%s"
    dockerfile   = "%s"
  }
}
