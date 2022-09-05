resource "docker_network" "test" {
  name = "tf-test"
  ipv6 = true

  ipam_config {
    subnet = "10.0.1.0/24"
  }

  ipam_config {
    subnet = "fd00::1/64"
  }
}
resource "docker_image" "foo" {
  name         = "nginx:latest"
  keep_locally = true
}
resource "docker_container" "foo" {
  name  = "tf-test"
  image = docker_image.foo.image_id
  networks_advanced {
    name         = docker_network.test.name
    ipv4_address = "10.0.1.123"
    ipv6_address = "fd00:0:0:0::123"
  }
}
