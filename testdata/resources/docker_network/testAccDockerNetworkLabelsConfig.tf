resource "docker_network" "foo" {
  name = "test_foo"
  labels {
    label = "com.docker.compose.network"
    value = "foo"
  }
  labels {
    label = "com.docker.compose.project"
    value = "test"
  }
}
