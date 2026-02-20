resource "docker_network" "foo" {
  name = "bar"

  driver   = "bridge"
  internal = true

  ipam_config {
    subnet  = "10.0.1.0/24"
  }

  labels {
    label = "com.docker.compose.network"
    value = "foo"
  }

  labels {
    label = "com.docker.compose.project"
    value = "test"
  }
}
