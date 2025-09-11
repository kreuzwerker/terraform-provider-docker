resource "docker_buildx_builder" "test_auto_recreate" {
  name = "test-auto-recreate-builder"
  docker_container {
    image = "moby/buildkit:v0.22.0"
  }
  use = true
  bootstrap = true
  auto_recreate = true
}

