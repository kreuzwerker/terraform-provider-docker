provider "docker" {
  alias = "private"
}

resource "docker_image" "foo_image" {
  provider = docker.private
  name     = "%s"
  build {
    context = "%s"
  }
}

resource "docker_registry_image" "foo" {
  provider             = docker.private
  name                 = docker_image.foo_image.name
  insecure_skip_verify = true
  keep_remotely        = true

  auth_config {
    address = "%s"
    username = "%s"
    password = "%s"
  }
}
