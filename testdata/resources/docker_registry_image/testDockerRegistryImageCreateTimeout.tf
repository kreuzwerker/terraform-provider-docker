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

  timeouts {
    create = "%s"
  }

  build {
    context    = "."
    dockerfile = "Dockerfile"
  }
}
