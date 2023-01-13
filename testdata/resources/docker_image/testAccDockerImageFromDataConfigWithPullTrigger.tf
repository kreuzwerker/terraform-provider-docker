data "docker_registry_image" "foobarbazoo" {
  name = "alpine:3.16.0"
}
resource "docker_image" "foobarbazoo" {
  name         = data.docker_registry_image.foobarbazoo.name
  pull_triggers = [data.docker_registry_image.foobarbazoo.sha256_digest]
}
