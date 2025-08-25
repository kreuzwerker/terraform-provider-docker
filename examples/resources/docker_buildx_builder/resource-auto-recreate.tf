# Example showing auto-recreation enabled for CI/CD environments
resource "docker_buildx_builder" "ci_builder" {
  name          = "ci-builder"
  driver        = "docker-container"
  bootstrap     = true
  auto_recreate = true # Automatically recreate if missing from Docker

  docker_container {
    image        = "moby/buildkit:latest"
    default_load = true
  }

  platform = ["linux/amd64", "linux/arm64"]
}

# Use the builder in a docker_image resource
resource "docker_image" "app" {
  name = "my-app:latest"

  build {
    context    = "."
    dockerfile = "Dockerfile"
    platform   = "linux/amd64,linux/arm64"

    # This builder will be auto-recreated if missing from Docker
    builder = "ci-builder"
  }

  depends_on = [docker_buildx_builder.ci_builder]
}
