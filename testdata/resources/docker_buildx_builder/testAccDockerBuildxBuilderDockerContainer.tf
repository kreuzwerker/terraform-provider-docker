resource "docker_buildx_builder" "foo" {
  name = "foo"
  docker_container {
    image = "docker:20.10.7"
  }
}
