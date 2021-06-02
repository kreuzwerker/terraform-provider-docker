provider "docker" {
  alias = "private"
  registry_auth {
    address = "%s"
  }
}
data "docker_registry_image" "foobar" {
  provider             = "docker.private"
  name                 = "%s"
  insecure_skip_verify = true
}