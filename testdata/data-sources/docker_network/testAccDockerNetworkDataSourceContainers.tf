resource "docker_network" "test" {
  name = "tf-test-docker-network-data-source"
  ipam_config {
    subnet  = "10.0.10.0/24"
    gateway = "10.0.10.1"
  }
}

resource "docker_image" "test" {
  name         = "nginx:latest"
  keep_locally = true
}

resource "docker_container" "test" {
  name  = "tf-test-docker-network-data-source"
  image = docker_image.test.image_id
  networks_advanced {
    name         = docker_network.test.name
    ipv4_address = "10.0.10.123"
  }
}

data "docker_network" "test" {
  name       = docker_network.test.name
  depends_on = [docker_container.test]
}
