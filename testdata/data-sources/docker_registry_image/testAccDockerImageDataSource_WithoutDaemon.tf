provider "docker" {
  alias = "private"
  disable_docker_daemon_check = true
  registry_auth {
    address = "%s"
  }
}
data "docker_registry_image" "foobar" {
  provider             = "docker.private"
  name                 = "%s"
  insecure_skip_verify = true
}