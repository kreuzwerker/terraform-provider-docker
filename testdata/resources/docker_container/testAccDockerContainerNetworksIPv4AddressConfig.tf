resource "docker_network" "test" {
  name = "tf-test"
  ipam_config {
    subnet = "10.0.1.0/24"
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
  }
}
