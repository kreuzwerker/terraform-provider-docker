provider "docker" {
  alias = "private"
  registry_auth {
    address = "127.0.0.1:15000"
  }
}
resource "docker_registry_image" "foo" {
  provider = "docker.private"
  name     = "127.0.0.1:15000/nonexistent:1.0"
}
