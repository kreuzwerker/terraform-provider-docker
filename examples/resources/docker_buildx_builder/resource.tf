resource "docker_buildx_builder" "example" {
  name   = "example-builder"
  driver = "docker-container"
  use    = true

  docker_container {
    image = "moby/buildkit:latest"
  }
}
