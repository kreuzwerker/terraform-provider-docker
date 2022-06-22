resource "docker_network" "test" {
  name = "tf-test"
  ipv6 = true
  ipam_config {
    subnet  = "fd00::1/64"
    gateway = "fd00:0:0:0::f"
  }
}
resource "docker_image" "foo" {
  name         = "busybox:1.35.0"
  keep_locally = true
}
resource "docker_container" "foo" {
  name  = "tf-test"
  image = docker_image.foo.latest
  networks_advanced {
    name         = docker_network.test.name
    ipv6_address = "fd00:0:0:0::123"
  }
}
