data "docker_registry_image" "foobarbaz" {
  name = "alpine:3.1"
}
resource "docker_image" "foobarbaz" {
  name          = data.docker_registry_image.foobarbaz.name
  pull_triggers = [data.docker_registry_image.foobarbaz.sha256_digest]
}
