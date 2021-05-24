resource "docker_network" "foo" {
  name = "bar"
  ipam_config {
    subnet = "10.0.1.0/24"
  }
}
