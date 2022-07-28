provider "docker" {
  alias = "private"
  registry_auth {
    address = "%s"
  }
}

resource "docker_registry_image" "outside_context" {
  provider             = "docker.private"
  name                 = "%s"
  insecure_skip_verify = true

  build {
    context    = "%s"
    dockerfile = "../Dockerfile"
  }
}
