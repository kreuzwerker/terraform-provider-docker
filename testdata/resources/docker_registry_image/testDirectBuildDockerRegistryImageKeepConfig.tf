provider "docker" {
  alias = "private"
  registry_auth {
    address = "%s"
  }
}

resource "docker_registry_image" "foo" {
  provider             = "docker.private"
  name                 = "%s"

  build {
    context = "%s"
  }
  insecure_skip_verify = true
  keep_remotely        = true
}
