data "docker_registry_image" "foobarbaz" {
  name = "alpine:3.16.0"
}
resource "docker_image" "foobarbaz" {
  name          = data.docker_registry_image.foobarbaz.name
  pull_triggers = [data.docker_registry_image.foobarbaz.sha256_digest]
}
