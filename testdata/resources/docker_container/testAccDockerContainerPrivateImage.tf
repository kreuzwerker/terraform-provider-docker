provider "docker" {
  alias = "private"
  registry_auth {
    address     = "%s"
    config_file = "%s"
  }
}

resource "docker_image" "foo" {
  provider     = "docker.private"
  name         = "%s"
  keep_locally = true
}

resource "docker_container" "foo" {
  provider = "docker.private"
  name     = "tf-test"
  image    = docker_image.foo.image_id
}
