resource "docker_buildx_builder" "example" {
  name          = "example-builder"
  driver        = "docker-container"
  bootstrap     = true
  auto_recreate = false # Default: will fail if builder is missing from Docker

  docker_container {
    image        = "moby/buildkit:latest"
    default_load = true
  }

  platform = ["linux/amd64", "linux/arm64"]
}
