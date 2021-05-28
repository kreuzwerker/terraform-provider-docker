resource "docker_network" "foo" {
  name = "bar"
  ipv6 = true
  ipam_config {
    subnet = "fd00::1/64"
  }
  # TODO mavogel: Would work but BC - #219
  #   ipam_config {
  #     subnet = "10.0.1.0/24"
  #   }
}
