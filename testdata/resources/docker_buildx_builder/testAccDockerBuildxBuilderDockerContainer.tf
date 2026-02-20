resource "docker_buildx_builder" "foo" {
  name = "foo"
  docker_container {
    image = "moby/buildkit:v0.22.0"
  }
  use = true
  bootstrap = true
}