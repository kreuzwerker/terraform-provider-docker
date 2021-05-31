data "docker_registry_image" "foobarbazoo" {
  name = "alpine:3.1"
}
resource "docker_image" "foobarbazoo" {
  name         = data.docker_registry_image.foobarbazoo.name
  pull_trigger = data.docker_registry_image.foobarbazoo.sha256_digest
}
